package command

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type Argument struct {
	Name        string
	Description string
	Optional    bool
	Type        ArgumentType
	Value       interface{}
	HasValue    bool

	// Prompt is the question asked for a missing required argument when
	// stdin is a terminal and --no-input was not passed. Without a prompt,
	// or when prompting is not possible, a missing argument is a usage
	// error.
	Prompt string

	// Validate checks each value of the argument before the command runs.
	// A returned error is reported as a usage error.
	Validate func(value string) error
}

// ArgumentType is an enum to define what arguments are present
type ArgumentType uint

const (
	ArgumentString ArgumentType = 0
	ArgumentInt    ArgumentType = 1 << iota
	ArgumentBool   ArgumentType = 2 << iota
	ArgumentList   ArgumentType = 3 << iota

	// ArgumentFile is a path to an existing file, or `-` for stdin.
	ArgumentFile ArgumentType = 32

	// ArgumentDuration is a duration such as `5m` or `1h30m`.
	ArgumentDuration ArgumentType = 64

	// ArgumentFloat is a floating point number.
	ArgumentFloat ArgumentType = 128
)

func (a Argument) BoolValue() bool {
	if a.Type != ArgumentBool {
		panic(fmt.Errorf("Unexpected argument type for %s when calling BoolValue()", a.Name))
	}

	return a.Value.(bool)
}

func (a Argument) IntValue() int {
	if a.Type != ArgumentInt {
		panic(fmt.Errorf("Unexpected argument type for %s when calling IntValue()", a.Name))
	}

	return a.Value.(int)
}

func (a Argument) StringValue() string {
	if a.Type != ArgumentString {
		panic(fmt.Errorf("Unexpected argument type for %s when calling StringValue()", a.Name))
	}

	return a.Value.(string)
}

// FileValue opens the file named by an ArgumentFile argument, or stdin when
// it is `-`. The caller must close it.
func (a Argument) FileValue() (io.ReadCloser, error) {
	if a.Type != ArgumentFile {
		panic(fmt.Errorf("Unexpected argument type for %s when calling FileValue()", a.Name))
	}

	return OpenInput(a.Value.(string))
}

// PathValue returns the path of an ArgumentFile argument.
func (a Argument) PathValue() string {
	if a.Type != ArgumentFile {
		panic(fmt.Errorf("Unexpected argument type for %s when calling PathValue()", a.Name))
	}

	return a.Value.(string)
}

func (a Argument) DurationValue() time.Duration {
	if a.Type != ArgumentDuration {
		panic(fmt.Errorf("Unexpected argument type for %s when calling DurationValue()", a.Name))
	}

	return a.Value.(time.Duration)
}

func (a Argument) FloatValue() float64 {
	if a.Type != ArgumentFloat {
		panic(fmt.Errorf("Unexpected argument type for %s when calling FloatValue()", a.Name))
	}

	return a.Value.(float64)
}

func (a Argument) ListValue() []string {
	if a.Type != ArgumentList {
		panic(fmt.Errorf("Unexpected argument type for %s when calling ListValue()", a.Name))
	}

	return a.Value.([]string)
}

func ArgumentAsString(arguments []Argument) string {
	argumentString := []string{}

	for _, argument := range arguments {
		suffix := ""
		if argument.Type == ArgumentList {
			suffix = "..."
		}

		if argument.Optional {
			argumentString = append(argumentString, fmt.Sprintf("[%s%s]", argument.Name, suffix))
		} else {
			argumentString = append(argumentString, fmt.Sprintf("<%s%s>", argument.Name, suffix))
		}
	}

	return strings.Join(argumentString, " ")
}

// ArgumentsString renders the arguments section of help output.
func ArgumentsString(arguments []Argument) string {
	return argumentsString(arguments, 0)
}

// argumentsString renders the arguments section of help output, wrapping
// descriptions to cols columns. Zero disables wrapping.
func argumentsString(arguments []Argument, cols int) string {
	maxlen := 0
	lines := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		line := ""

		suffix := ""
		if argument.Type == ArgumentList {
			suffix = "..."
		}

		if argument.Optional {
			line = fmt.Sprintf("  [%s%s]", argument.Name, suffix)
		} else {
			line = fmt.Sprintf("  <%s%s>", argument.Name, suffix)
		}

		switch argument.Type {
		case ArgumentString:
			line += " string"
		case ArgumentInt:
			line += " int"
		case ArgumentBool:
			line += " bool"
		case ArgumentFile:
			line += " file"
		case ArgumentDuration:
			line += " duration"
		case ArgumentFloat:
			line += " float"
		}

		line += "\x00"
		if len(line) > maxlen {
			maxlen = len(line)
		}

		line += argument.Description
		lines = append(lines, line)
	}

	buf := new(bytes.Buffer)
	for _, line := range lines {
		sidx := strings.Index(line, "\x00")
		spacing := strings.Repeat(" ", maxlen-sidx)
		// maxlen + 2 comes from + 1 for the \x00 and + 1 for the (deliberate) off-by-one in maxlen-sidx
		fmt.Fprintln(buf, line[:sidx], spacing, wrap(maxlen+2, cols, line[sidx+1:]))
	}

	return strings.TrimRight(buf.String(), "\n")
}

// Splits the string `s` on whitespace into an initial substring up to
// `i` runes in length and the remainder. Will go `slop` over `i` if
// that encompasses the entire string (which allows the caller to
// avoid short orphan words on the final line).
func wrapN(i, slop int, s string) (string, string) {
	if i+slop > len(s) {
		return s, ""
	}

	w := strings.LastIndexAny(s[:i], " \t\n")
	if w <= 0 {
		return s, ""
	}
	nlPos := strings.LastIndex(s[:i], "\n")
	if nlPos > 0 && nlPos < w {
		return s[:nlPos], s[nlPos+1:]
	}
	return s[:w], s[w+1:]
}

// Wraps the string `s` to a maximum width `w` with leading indent
// `i`. The first line is not indented (this is assumed to be done by
// caller). Pass `w` == 0 to do no wrapping
func wrap(i, w int, s string) string {
	if w == 0 {
		return strings.Replace(s, "\n", "\n"+strings.Repeat(" ", i), -1)
	}

	// space between indent i and end of line width w into which
	// we should wrap the text.
	wrap := w - i

	var r, l string

	// Not enough space for sensible wrapping. Wrap as a block on
	// the next line instead.
	if wrap < 24 {
		i = 16
		wrap = w - i
		r += "\n" + strings.Repeat(" ", i)
	}
	// If still not enough space then don't even try to wrap.
	if wrap < 24 {
		return strings.Replace(s, "\n", r, -1)
	}

	// Try to avoid short orphan words on the final line, by
	// allowing wrapN to go a bit over if that would fit in the
	// remainder of the line.
	slop := 5
	wrap = wrap - slop

	// Handle first line, which is indented by the caller (or the
	// special case above)
	l, s = wrapN(wrap, slop, s)
	r = r + strings.Replace(l, "\n", "\n"+strings.Repeat(" ", i), -1)

	// Now wrap the rest
	for s != "" {
		var t string

		t, s = wrapN(wrap, slop, s)
		r = r + "\n" + strings.Repeat(" ", i) + strings.Replace(t, "\n", "\n"+strings.Repeat(" ", i), -1)
	}

	return r

}

func ParseArguments(args []string, arguments []Argument) (map[string]Argument, error) {
	returnArguments := map[string]Argument{}
	if err := validateArguments(arguments); err != nil {
		return returnArguments, err
	}

	maxArgs := len(arguments)
	minArgs := 0
	for _, argument := range arguments {
		if !argument.Optional {
			minArgs++
		}
	}

	checkMaxArgs := true
	if len(arguments) > 0 {
		if arguments[len(arguments)-1].Type == ArgumentList {
			checkMaxArgs = false
		}
	}

	argumentWord := "argument"
	if maxArgs != 1 {
		argumentWord = "arguments"
	}
	errorMessage := fmt.Sprintf("This command requires %d", minArgs)
	if minArgs != maxArgs {
		errorMessage = fmt.Sprintf("%s and at most %d %s", errorMessage, maxArgs, argumentWord)
	} else {
		errorMessage = fmt.Sprintf("%s %s", errorMessage, argumentWord)
	}

	if len(args) == 0 {
		if len(arguments) == 0 {
			return returnArguments, nil
		}

		if !arguments[0].Optional {
			return returnArguments, fmt.Errorf("%s: %s", errorMessage, ArgumentAsString(arguments))
		}
	}

	if len(args) < minArgs || (len(args) > maxArgs && checkMaxArgs) {
		argumentWord := "argument"
		if len(args) != 1 {
			argumentWord = "arguments"
		}
		return returnArguments, fmt.Errorf("%s, %d %s given: %s", errorMessage, len(args), argumentWord, ArgumentAsString(arguments))
	}

	hasListArgument := false
	listIndex := 0
	for i, value := range args {
		index := i
		if hasListArgument {
			index = listIndex
		}

		if validate := arguments[index].Validate; validate != nil {
			if err := validate(value); err != nil {
				return returnArguments, fmt.Errorf("Invalid value for argument %s: %w", arguments[index].Name, err)
			}
		}

		if hasListArgument {
			arguments[listIndex].HasValue = true
			arguments[listIndex].Value = append(arguments[listIndex].Value.([]string), value)
			continue
		}

		arguments[i].HasValue = true
		if arguments[i].Type == ArgumentList {
			hasListArgument = true
			listIndex = i
			arguments[i].Value = []string{value}
			continue
		}

		parsed, err := parseArgumentValue(arguments[i], value)
		if err != nil {
			return returnArguments, err
		}
		arguments[i].Value = parsed
	}

	for _, argument := range arguments {
		if argument.Value == nil {
			switch argument.Type {
			case ArgumentBool:
				argument.Value = false
			case ArgumentInt:
				argument.Value = 0
			case ArgumentList:
				argument.Value = []string{}
			case ArgumentString, ArgumentFile:
				argument.Value = ""
			case ArgumentDuration:
				argument.Value = time.Duration(0)
			case ArgumentFloat:
				argument.Value = float64(0)
			}
			argument.HasValue = false
		}
		returnArguments[argument.Name] = argument
	}

	return returnArguments, nil
}

// parseArgumentValue converts a single value to the type of the argument.
func parseArgumentValue(argument Argument, value string) (interface{}, error) {
	invalid := fmt.Errorf("Invalid value for argument %s", argument.Name)

	switch argument.Type {
	case ArgumentInt:
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return nil, invalid
		}
		return intValue, nil
	case ArgumentBool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return nil, invalid
		}
		return boolValue, nil
	case ArgumentDuration:
		duration, err := time.ParseDuration(value)
		if err != nil {
			return nil, fmt.Errorf("Invalid value for argument %s: expected a duration such as 30s or 5m", argument.Name)
		}
		return duration, nil
	case ArgumentFloat:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, invalid
		}
		return floatValue, nil
	case ArgumentFile:
		if value != "-" {
			info, err := os.Stat(value)
			if err != nil {
				return nil, fmt.Errorf("Invalid value for argument %s: no such file %s", argument.Name, value)
			}
			if info.IsDir() {
				return nil, fmt.Errorf("Invalid value for argument %s: %s is a directory", argument.Name, value)
			}
		}
		return value, nil
	default:
		return value, nil
	}
}

func validateArguments(arguments []Argument) error {
	reachedOptional := false
	reachedList := false
	listArgument := ""
	for _, arg := range arguments {
		if reachedOptional {
			if !arg.Optional {
				return fmt.Errorf("Argument %s must be placed before all optional arguments", arg.Name)
			}
		} else if arg.Optional {
			reachedOptional = true
		}

		if reachedList {
			return fmt.Errorf("List Argument %s must be placed after all other arguments", listArgument)
		} else if arg.Type == ArgumentList {
			listArgument = arg.Name
			reachedList = true
		}
	}
	return nil
}
