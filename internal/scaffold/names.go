package scaffold

import (
	"go/token"
	"strings"
)

// words splits a command or flag name into its words, such as
// `lollipop dry-run` into lollipop, dry and run.
func words(name string) []string {
	return strings.FieldsFunc(name, func(r rune) bool {
		return r == ' ' || r == '-' || r == '_'
	})
}

func title(word string) string {
	if word == "" {
		return word
	}
	return strings.ToUpper(word[:1]) + word[1:]
}

// TypeName returns the name of the struct for a command, such as
// LollipopEatCommand for `lollipop eat`.
func TypeName(command string) string {
	var b strings.Builder
	for _, word := range words(command) {
		b.WriteString(title(word))
	}
	b.WriteString("Command")
	return b.String()
}

// FileName returns the file a command is written to, relative to the
// commands directory, such as lollipop-eat.go for `lollipop eat`.
//
// Words are joined with dashes rather than underscores, so that commands
// such as `server linux` or `unit test` are not mistaken for build
// constrained or test files.
func FileName(command string) string {
	return strings.Join(words(command), "-") + ".go"
}

// FieldName returns the struct field that holds a flag value, such as
// globalString for `global-string`.
func FieldName(flag string) string {
	parts := words(flag)
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(parts[0])
	for _, word := range parts[1:] {
		b.WriteString(title(word))
	}
	name := b.String()

	// A field named meta would hide the meta method of command.Meta, which
	// every command needs, and keywords are not valid field names.
	if name == "meta" || token.IsKeyword(name) {
		name += "Value"
	}
	return name
}

// flagSetsConst returns the name of the constant holding the flag sets of
// a command, such as lollipopEatFlags for `lollipop eat`.
func flagSetsConst(command string) string {
	parts := words(command)
	var b strings.Builder
	b.WriteString(parts[0])
	for _, word := range parts[1:] {
		b.WriteString(title(word))
	}
	b.WriteString("Flags")
	return b.String()
}
