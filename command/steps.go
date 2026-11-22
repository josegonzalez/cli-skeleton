package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// DefaultTailLines is how many recent output lines a running step shows.
const DefaultTailLines = 4

// frameInterval is how often running steps are redrawn.
const frameInterval = 100 * time.Millisecond

// Progress output modes.
const (
	// progressFancy redraws running steps in place, with a spinner and a
	// live tail of their output. It is used when stderr is a terminal.
	progressFancy = "fancy"

	// progressPlain prints one line when a step starts and one when it
	// ends. It is used when stderr is not a terminal, such as in CI.
	progressPlain = "plain"

	// progressVerbose prints every line of step output. It is used under
	// --debug and <APP>_VERBOSE.
	progressVerbose = "verbose"

	// progressQuiet prints nothing unless a step fails. It is used under
	// --quiet.
	progressQuiet = "quiet"
)

// timeNow returns the current time. Tests replace it with a fake clock.
var timeNow = time.Now

// WithTailLines sets how many recent output lines a running step shows on
// a terminal. Zero shows only the spinner.
func WithTailLines(n int) Option {
	return func(o *mainOptions) {
		o.tailLines = &n
	}
}

// progressState is shared by every command of a run, so steps share one
// renderer and one log file.
type progressState struct {
	mu       sync.Mutex
	renderer *renderer
	log      *stepLog
}

// close closes the log file.
func (p *progressState) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.log != nil {
		p.log.close()
	}
}

// progress returns the progress state of the run, creating it when the Meta
// was not created by Main.
func (m *Meta) progress() *progressState {
	if m.progressState == nil {
		m.progressState = &progressState{}
	}
	return m.progressState
}

// progressRenderer returns the renderer shared by the run.
func (m *Meta) progressRenderer() *renderer {
	p := m.progress()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.renderer == nil {
		p.renderer = newRenderer(m)
	}
	return p.renderer
}

// progressMode picks how steps are shown.
func (m *Meta) progressMode() string {
	switch {
	case m.progressModeOverride != "":
		return m.progressModeOverride
	case m.IsQuiet():
		return progressQuiet
	case m.IsVerbose():
		return progressVerbose
	case m.IsStderrTerminal() && os.Getenv("TERM") != "dumb":
		return progressFancy
	default:
		return progressPlain
	}
}

// renderer draws steps, spinners and progress bars on stderr.
type renderer struct {
	m      *Meta
	w      io.Writer
	mode   string
	color  bool
	utf8   bool
	width  func() int
	log    *stepLog
	spin   []string
	mu     sync.Mutex
	active []*Step
	frame  int
	drawn  bool
	stop   chan struct{}
	unhook func()
}

func newRenderer(m *Meta) *renderer {
	w := m.stderr
	if w == nil {
		w = os.Stderr
	}

	r := &renderer{
		m:     m,
		w:     w,
		mode:  m.progressMode(),
		color: m.stderrColor(),
		utf8:  supportsUTF8(),
		width: func() int {
			f := m.stderrFile
			if f == nil {
				if file, ok := m.stderr.(*os.File); ok {
					f = file
				}
			}
			return terminalWidth(f)
		},
	}

	r.spin = []string{"|", "/", "-", "\\"}
	if r.utf8 {
		r.spin = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	}
	return r
}

// stepLog writes every line of step output, with timestamps, to a file.
type stepLog struct {
	mu     sync.Mutex
	w      io.Writer
	closer io.Closer
	path   string
}

func (l *stepLog) printf(format string, args ...interface{}) {
	if l == nil || l.w == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.w, "%s %s\n", timeNow().UTC().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

func (l *stepLog) close() {
	if l != nil && l.closer != nil {
		l.closer.Close()
		l.closer = nil
		l.w = nil
	}
}

// stepLogFile returns the log of the run, opening it on first use. The log
// is written to <APP>_DEBUG_LOG when set, and otherwise to a timestamped
// file in StateDir()/logs, falling back to TMPDIR.
func (m *Meta) stepLogFile() *stepLog {
	p := m.progress()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.log != nil {
		return p.log
	}

	if m.debugLog != nil {
		path := ""
		if name := AppEnv(m.appName, "DEBUG_LOG"); name != "" {
			path = os.Getenv(name)
		}
		p.log = &stepLog{w: m.debugLog, path: path}
		return p.log
	}

	name := commandPath(m.commandName)
	if name == "" {
		name = "run"
	}
	prefix := m.appName
	if prefix == "" {
		prefix = "cli"
	}
	file := fmt.Sprintf("%s-%s-%s.log", prefix, name, timeNow().Format("20060102-150405"))

	var dirs []string
	if dir, err := StateDir(m.appName); err == nil && m.appName != "" {
		dirs = append(dirs, filepath.Join(dir, "logs"))
	}
	dirs = append(dirs, os.TempDir())

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			continue
		}
		path := filepath.Join(dir, file)
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			continue
		}
		p.log = &stepLog{w: f, closer: f, path: path}
		return p.log
	}

	p.log = &stepLog{}
	return p.log
}

// Step is a unit of work shown with progress output. Write the step's log
// output to it, run external commands with Exec, and report a short result
// with Result.
type Step struct {
	r         *renderer
	title     string
	index     int
	total     int
	tailLines int
	start     time.Time

	tail    []string
	output  []string
	partial string
	results []string
	done    bool

	bar *barState
}

// barState tracks a progress bar.
type barState struct {
	current     int64
	total       int64
	lastPercent int
}

// start begins showing a new step.
func (r *renderer) start(title string, index int, total int, tailLines int) *Step {
	s := &Step{
		r:         r,
		title:     title,
		index:     index,
		total:     total,
		tailLines: tailLines,
		start:     timeNow(),
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.log == nil {
		r.log = r.m.stepLogFile()
	}
	r.log.printf("==> step start %s", s.plainLabel())

	r.active = append(r.active, s)
	if len(r.active) == 1 {
		r.unhook = r.m.onInterrupt(r.interrupted)
	}

	switch r.mode {
	case progressFancy:
		if len(r.active) == 1 {
			fmt.Fprint(r.w, "\x1b[?25l")
			r.startTicker()
		}
		r.drawLocked()
	case progressPlain, progressVerbose:
		fmt.Fprintln(r.w, r.stepLine(s))
	}
	return s
}

// startTicker redraws the running steps until they finish.
func (r *renderer) startTicker() {
	stop := make(chan struct{})
	r.stop = stop
	go func() {
		ticker := time.NewTicker(frameInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				r.mu.Lock()
				r.drawLocked()
				r.mu.Unlock()
			case <-stop:
				return
			}
		}
	}()
}

// label returns the step counter and title, such as `[1/3] Installing`.
func (s *Step) label(color bool) string {
	title := s.title
	if color {
		title = "\x1b[1m" + title + "\x1b[0m"
	}
	if s.total <= 0 {
		return title
	}
	return muted(fmt.Sprintf("[%d/%d]", s.index, s.total), color) + " " + title
}

// plainLabel returns the label without colors.
func (s *Step) plainLabel() string {
	return s.label(false)
}

// muted dims text when color is enabled.
func muted(text string, color bool) string {
	return colorize(text, colorDarkGray, !color)
}

// stepLine is the line printed for a step in plain and verbose mode, and
// for a finished step in fancy mode.
func (r *renderer) stepLine(s *Step) string {
	return colorize("----->", colorCyan, !r.color) + " " + s.label(r.color)
}

// Write adds output to the step. Complete lines are written to the log, to
// the live tail and, in verbose mode, to stderr.
func (s *Step) Write(p []byte) (int, error) {
	s.r.mu.Lock()
	defer s.r.mu.Unlock()

	if s.done {
		return len(p), nil
	}

	s.partial += string(p)
	for {
		i := strings.IndexByte(s.partial, '\n')
		if i < 0 {
			break
		}
		s.addLineLocked(s.partial[:i])
		s.partial = s.partial[i+1:]
	}
	return len(p), nil
}

// Printf writes a formatted line of output to the step.
func (s *Step) Printf(format string, args ...interface{}) {
	fmt.Fprint(s, ensureNewline(fmt.Sprintf(format, args...)))
}

// Result records a short description of what the step did, shown when it
// finishes, such as "installed v1.2.3".
func (s *Step) Result(text string) {
	s.r.mu.Lock()
	defer s.r.mu.Unlock()
	s.results = append(s.results, text)
}

var escapeSequence = regexp.MustCompile("\x1b\\[[0-9;?]*[A-Za-z]")

// sanitize removes escape sequences and control characters from a line of
// output, keeping only what a carriage return would leave on screen.
func sanitize(line string) string {
	line = strings.TrimRight(line, "\r")
	if i := strings.LastIndexByte(line, '\r'); i >= 0 {
		line = line[i+1:]
	}
	line = strings.ReplaceAll(line, "\t", "    ")
	line = escapeSequence.ReplaceAllString(line, "")
	return strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, line)
}

func (s *Step) addLineLocked(raw string) {
	r := s.r
	line := sanitize(raw)
	r.log.printf("%s", strings.TrimRight(raw, "\r"))

	elapsed := formatDuration(timeNow().Sub(s.start))
	s.output = append(s.output, line)
	s.tail = append(s.tail, "+"+elapsed+" "+line)
	if len(s.tail) > s.tailLines {
		s.tail = s.tail[len(s.tail)-s.tailLines:]
	}

	if r.mode == progressVerbose {
		prefix := ""
		if len(r.active) > 1 {
			prefix = "[" + s.title + "] "
		}
		fmt.Fprintln(r.w, "       "+muted("+"+elapsed, r.color)+" "+prefix+line)
	}
}

// Done finishes the step. On success, it shows the step as finished with
// its duration and result. On failure, it shows the step's output, the log
// path and where to get help, and returns a StepError, which Execute does
// not print again. When the run was interrupted, the error is returned as
// is.
func (s *Step) Done(err error) error {
	r := s.r
	r.mu.Lock()
	defer r.mu.Unlock()

	if s.done {
		return err
	}
	if s.partial != "" {
		s.addLineLocked(s.partial)
		s.partial = ""
	}
	s.done = true

	duration := formatDuration(timeNow().Sub(s.start))
	interrupted := errors.Is(err, context.Canceled) || (r.m.Context != nil && r.m.Context.Err() != nil && err != nil)

	status := "ok"
	if err != nil {
		status = err.Error()
	}
	r.log.printf("==> step end %s (%s, %s)", s.plainLabel(), duration, status)

	if r.mode == progressFancy {
		r.clearLocked()
	}
	r.removeLocked(s)

	switch {
	case err == nil:
		r.printSuccessLocked(s, duration)
	case interrupted:
	default:
		r.printFailureLocked(s, duration, err)
	}

	if r.mode == progressFancy {
		if len(r.active) == 0 {
			r.stopLocked()
		} else {
			r.drawLocked()
		}
	} else if len(r.active) == 0 {
		r.unhookLocked()
	}

	if err == nil || interrupted {
		return err
	}
	return &StepError{Title: s.title, Err: err}
}

func (r *renderer) printSuccessLocked(s *Step, duration string) {
	result := strings.Join(s.results, ", ")
	switch r.mode {
	case progressFancy:
		fmt.Fprintln(r.w, r.stepLine(s)+" "+muted("("+duration+")", r.color))
		if result != "" {
			fmt.Fprintln(r.w, "       "+muted(result, r.color))
		}
	case progressPlain, progressVerbose:
		if result != "" {
			result += " "
		}
		fmt.Fprintln(r.w, "       "+muted(result+"("+duration+")", r.color))
	}
}

func (r *renderer) printFailureLocked(s *Step, duration string, err error) {
	bang := colorize(" !     ", colorRed, !r.color)
	fmt.Fprintln(r.w, bang+colorize(fmt.Sprintf("%s failed after %s. Output from this step:", s.plainLabel(), duration), colorRed, !r.color))
	for _, line := range s.output {
		fmt.Fprintln(r.w, "       "+line)
	}
	fmt.Fprintln(r.w, bang+colorize(err.Error(), colorRed, !r.color))
	if r.log != nil && r.log.path != "" {
		fmt.Fprintln(r.w, bang+"Full log: "+r.log.path)
	}
	if r.m.supportURL != "" {
		fmt.Fprintln(r.w, bang+"Help: "+r.m.supportURL)
	}
}

func (r *renderer) removeLocked(s *Step) {
	for i, active := range r.active {
		if active == s {
			r.active = append(r.active[:i], r.active[i+1:]...)
			return
		}
	}
}

func (r *renderer) stopLocked() {
	if r.stop != nil {
		close(r.stop)
		r.stop = nil
	}
	fmt.Fprint(r.w, "\x1b[?25h")
	r.unhookLocked()
}

func (r *renderer) unhookLocked() {
	if r.unhook != nil {
		r.unhook()
		r.unhook = nil
	}
}

// interrupted runs on the first interrupt: it clears the running steps,
// restores the cursor and points to the log.
func (r *renderer) interrupted() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.mode == progressFancy {
		r.clearLocked()
		if r.stop != nil {
			close(r.stop)
			r.stop = nil
		}
		fmt.Fprint(r.w, "\x1b[?25h")
	}
	for _, s := range r.active {
		s.done = true
		r.log.printf("==> step interrupted %s", s.plainLabel())
	}
	r.active = nil

	if r.log != nil && r.log.path != "" {
		fmt.Fprintln(r.w, colorize(" !     ", colorRed, !r.color)+"Full log: "+r.log.path)
	}
}

// clearLocked erases the frame drawn for the running steps.
func (r *renderer) clearLocked() {
	if r.drawn {
		fmt.Fprint(r.w, "\r\x1b[J")
		r.drawn = false
	}
}

// drawLocked redraws the running steps in place: a spinner, the step label
// and elapsed time, followed by the recent output of each step. The cursor
// is left at the start of the frame, so the next draw overwrites it.
func (r *renderer) drawLocked() {
	if r.mode != progressFancy || len(r.active) == 0 {
		return
	}

	width := r.width()
	if width <= 0 {
		width = 80
	}
	spinner := r.spin[r.frame%len(r.spin)]
	r.frame++

	var lines []string
	for _, s := range r.active {
		elapsed := formatDuration(timeNow().Sub(s.start))
		head := colorize("----"+spinner, colorCyan, !r.color) + "  " + s.label(r.color)
		visible := 6 + displayWidth(s.plainLabel())
		if s.bar != nil {
			bar := s.bar.render(timeNow().Sub(s.start), r.utf8)
			head += " " + bar
			visible += 1 + displayWidth(bar)
		}
		suffix := " (" + elapsed + ")"
		if visible+len(suffix) < width {
			head += " " + muted("("+elapsed+")", r.color)
		}
		lines = append(lines, head+"\x1b[K")

		for _, line := range s.tail {
			lines = append(lines, "       "+muted(truncate(line, width-8, r.utf8), r.color)+"\x1b[K")
		}
	}

	frame := "\r" + strings.Join(lines, "\n") + "\x1b[J"
	if len(lines) > 1 {
		frame += fmt.Sprintf("\x1b[%dA", len(lines)-1)
	}
	frame += "\r"
	fmt.Fprint(r.w, frame)
	r.drawn = true
}

// displayWidth returns the number of runes in text.
func displayWidth(text string) int {
	return len([]rune(text))
}

// truncate shortens text to at most max runes, ending with an ellipsis.
func truncate(text string, max int, utf8 bool) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	ellipsis := "…"
	if !utf8 {
		ellipsis = "."
	}
	return string(runes[:max-1]) + ellipsis
}

// formatDuration formats durations the way steps show them: 0.4s, 12s,
// 3m05s.
func formatDuration(d time.Duration) string {
	switch {
	case d < 10*time.Second:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	default:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
}

// StepError is returned when a step fails. The failure has already been
// shown, so Execute only sets the exit code.
type StepError struct {
	// Title is the title of the failed step.
	Title string

	// Err is the error the step returned.
	Err error
}

func (e *StepError) Error() string {
	return e.Title + ": " + e.Err.Error()
}

func (e *StepError) Unwrap() error {
	return e.Err
}

// Steps shows a numbered sequence of steps, such as `[1/3] Installing`.
type Steps struct {
	m      *Meta
	r      *renderer
	total  int
	mu     sync.Mutex
	next   int
	start  time.Time
	timing []stepTiming
}

type stepTiming struct {
	title    string
	duration time.Duration
}

// Steps returns a sequence of total steps. Run each step with Run:
//
//	steps := c.Steps(2)
//	err := steps.Run("Fetching lollipops", func(step *command.Step) error {
//		return step.Exec(exec.Command("curl", "-fsSLO", url))
//	})
//
// On a terminal, the running step shows a spinner and its recent output,
// and collapses to a single line when it succeeds. Outside a terminal, a
// line is printed when each step starts and ends. Under --debug or
// <APP>_VERBOSE every line of output is shown, and under --quiet nothing is
// shown unless a step fails. A failed step shows its output and the path
// of the log, which always holds the full output.
func (m *Meta) Steps(total int) *Steps {
	return &Steps{m: m, r: m.progressRenderer(), total: total, start: timeNow()}
}

// Run runs fn as the next step. The step is an io.Writer for fn's output.
func (s *Steps) Run(title string, fn func(step *Step) error) (err error) {
	s.mu.Lock()
	s.next++
	index := s.next
	s.mu.Unlock()

	step := s.r.start(title, index, s.total, s.m.tailLineCount())
	defer func() {
		if r := recover(); r != nil {
			step.Done(fmt.Errorf("panic: %v", r))
			panic(r)
		}
	}()

	err = step.Done(fn(step))

	s.mu.Lock()
	s.timing = append(s.timing, stepTiming{title: title, duration: timeNow().Sub(step.start)})
	s.mu.Unlock()
	return err
}

// Summary prints how long the steps took, and the three slowest steps.
func (s *Steps) Summary() {
	s.mu.Lock()
	timing := append([]stepTiming{}, s.timing...)
	s.mu.Unlock()

	sort.SliceStable(timing, func(i, j int) bool { return timing[i].duration > timing[j].duration })
	if len(timing) > 3 {
		timing = timing[:3]
	}

	message := "Finished in " + formatDuration(timeNow().Sub(s.start))
	if len(timing) > 0 {
		slowest := make([]string, 0, len(timing))
		for _, t := range timing {
			slowest = append(slowest, t.title+" "+formatDuration(t.duration))
		}
		message += ". Slowest steps: " + strings.Join(slowest, ", ")
	}
	s.m.Ui.Info(message + ".")
}

// tailLineCount returns how many output lines a running step shows.
func (m *Meta) tailLineCount() int {
	if m.tailLines == nil {
		return DefaultTailLines
	}
	return max(*m.tailLines, 0)
}

// Spinner shows a spinner with a title until Done is called on the
// returned step. Write output to the step to log it; it is not shown on a
// terminal unless the step fails.
func (m *Meta) Spinner(title string) *Step {
	return m.progressRenderer().start(title, 0, 0, 0)
}

// ProgressBar shows a progress bar for work measured in units, such as
// bytes or files, until Done is called. On a terminal it shows the count,
// percent and estimated time remaining; elsewhere it prints a line every
// 10%.
type ProgressBar struct {
	*Step
}

// ProgressBar returns a progress bar for total units of work.
func (m *Meta) ProgressBar(title string, total int64) *ProgressBar {
	r := m.progressRenderer()
	s := r.start(title, 0, 0, m.tailLineCount())
	r.mu.Lock()
	s.bar = &barState{total: total, lastPercent: -1}
	r.mu.Unlock()
	return &ProgressBar{Step: s}
}

// Add advances the bar by n units.
func (b *ProgressBar) Add(n int64) {
	b.r.mu.Lock()
	current := b.bar.current + n
	b.r.mu.Unlock()
	b.Set(current)
}

// Set moves the bar to n units.
func (b *ProgressBar) Set(n int64) {
	r := b.r
	r.mu.Lock()
	defer r.mu.Unlock()

	if b.done {
		return
	}
	b.bar.current = min(max(n, 0), b.bar.total)

	if r.mode == progressPlain || r.mode == progressVerbose {
		percent := b.bar.percent()
		if step := percent / 10 * 10; step > b.bar.lastPercent {
			b.bar.lastPercent = step
			fmt.Fprintln(r.w, "       "+muted(fmt.Sprintf("%s %d%% (%d/%d)", b.title, step, b.bar.current, b.bar.total), r.color))
		}
	}
}

func (b *barState) percent() int {
	if b.total <= 0 {
		return 100
	}
	return int(b.current * 100 / b.total)
}

// render draws the bar, percent, count and estimated time remaining.
func (b *barState) render(elapsed time.Duration, utf8 bool) string {
	const width = 20
	filled := 0
	if b.total > 0 {
		filled = int(b.current * width / b.total)
	}

	full, empty := "=", "-"
	if utf8 {
		full, empty = "█", "░"
	}
	bar := strings.Repeat(full, filled) + strings.Repeat(empty, width-filled)

	text := fmt.Sprintf("%s %3d%% %d/%d", bar, b.percent(), b.current, b.total)
	if b.current > 0 && b.current < b.total {
		remaining := time.Duration(float64(elapsed) * float64(b.total-b.current) / float64(b.current))
		text += " ETA " + formatDuration(remaining)
	}
	return text
}

// Countdown warns the user and waits before continuing, showing the time
// left on a terminal. It returns right away under --force, and returns the
// context error when interrupted.
func (m *Meta) Countdown(message string, seconds int) error {
	if m.flags.force || seconds <= 0 {
		return nil
	}

	ctx := m.context()
	r := m.progressRenderer()
	m.Ui.Warn(message)

	if r.mode != progressFancy {
		m.Ui.Warn(fmt.Sprintf("Continuing in %ds...", seconds))
		select {
		case <-time.After(time.Duration(seconds) * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	ellipsis := "..."
	if r.utf8 {
		ellipsis = "…"
	}

	fmt.Fprint(r.w, "\x1b[?25l")
	defer fmt.Fprint(r.w, "\x1b[?25h")
	for remaining := seconds; remaining > 0; remaining-- {
		fmt.Fprintf(r.w, "\r%s\x1b[K", colorize(fmt.Sprintf("Continuing in %ds%s", remaining, ellipsis), colorYellow, !r.color))
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			fmt.Fprint(r.w, "\r\x1b[K")
			return ctx.Err()
		}
	}
	fmt.Fprint(r.w, "\r\x1b[K")
	m.Ui.Warn(fmt.Sprintf("Waited %ds.", seconds))
	return nil
}
