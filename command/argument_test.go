package command

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseArguments(t *testing.T) {
	tests := []struct {
		name      string
		arguments []Argument
		args      []string
		want      map[string]interface{}
		hasValue  map[string]bool
		wantErr   string
	}{
		{
			name:      "no arguments and no args",
			arguments: []Argument{},
			args:      []string{},
			want:      map[string]interface{}{},
		},
		{
			name:      "required string",
			arguments: []Argument{{Name: "name", Type: ArgumentString}},
			args:      []string{"lollipop"},
			want:      map[string]interface{}{"name": "lollipop"},
			hasValue:  map[string]bool{"name": true},
		},
		{
			name:      "missing required string",
			arguments: []Argument{{Name: "name", Type: ArgumentString}},
			args:      []string{},
			wantErr:   "This command requires 1 argument: <name>",
		},
		{
			name:      "optional string not given",
			arguments: []Argument{{Name: "speed", Type: ArgumentString, Optional: true}},
			args:      []string{},
			want:      map[string]interface{}{"speed": ""},
			hasValue:  map[string]bool{"speed": false},
		},
		{
			name:      "int",
			arguments: []Argument{{Name: "count", Type: ArgumentInt}},
			args:      []string{"3"},
			want:      map[string]interface{}{"count": 3},
		},
		{
			name:      "invalid int",
			arguments: []Argument{{Name: "count", Type: ArgumentInt}},
			args:      []string{"three"},
			wantErr:   "Invalid value for argument count",
		},
		{
			name:      "bool",
			arguments: []Argument{{Name: "enabled", Type: ArgumentBool}},
			args:      []string{"true"},
			want:      map[string]interface{}{"enabled": true},
		},
		{
			name:      "bool false",
			arguments: []Argument{{Name: "enabled", Type: ArgumentBool}},
			args:      []string{"0"},
			want:      map[string]interface{}{"enabled": false},
		},
		{
			name:      "invalid bool",
			arguments: []Argument{{Name: "enabled", Type: ArgumentBool}},
			args:      []string{"maybe"},
			wantErr:   "Invalid value for argument enabled",
		},
		{
			name:      "optional bool not given",
			arguments: []Argument{{Name: "enabled", Type: ArgumentBool, Optional: true}},
			args:      []string{},
			want:      map[string]interface{}{"enabled": false},
		},
		{
			name: "list collects the rest",
			arguments: []Argument{
				{Name: "first", Type: ArgumentString},
				{Name: "rest", Type: ArgumentList},
			},
			args: []string{"a", "b", "c"},
			want: map[string]interface{}{"first": "a", "rest": []string{"b", "c"}},
		},
		{
			name: "optional list not given",
			arguments: []Argument{
				{Name: "first", Type: ArgumentString},
				{Name: "rest", Type: ArgumentList, Optional: true},
			},
			args: []string{"a"},
			want: map[string]interface{}{"first": "a", "rest": []string{}},
		},
		{
			name:      "too many args",
			arguments: []Argument{{Name: "name", Type: ArgumentString}},
			args:      []string{"a", "b"},
			wantErr:   "This command requires 1 argument, 2 arguments given: <name>",
		},
		{
			name: "between min and max",
			arguments: []Argument{
				{Name: "name", Type: ArgumentString},
				{Name: "speed", Type: ArgumentString, Optional: true},
			},
			args:    []string{"a", "b", "c"},
			wantErr: "This command requires 1 and at most 2 arguments, 3 arguments given: <name> [speed]",
		},
		{
			name: "required after optional",
			arguments: []Argument{
				{Name: "speed", Type: ArgumentString, Optional: true},
				{Name: "name", Type: ArgumentString},
			},
			args:    []string{"a"},
			wantErr: "Argument name must be placed before all optional arguments",
		},
		{
			name: "list not last",
			arguments: []Argument{
				{Name: "rest", Type: ArgumentList},
				{Name: "name", Type: ArgumentString},
			},
			args:    []string{"a"},
			wantErr: "List Argument rest must be placed after all other arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseArguments(tt.args, tt.arguments)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("expected %d arguments, got %d", len(tt.want), len(got))
			}
			for name, want := range tt.want {
				if !reflect.DeepEqual(got[name].Value, want) {
					t.Errorf("argument %s: expected %#v, got %#v", name, want, got[name].Value)
				}
			}
			for name, want := range tt.hasValue {
				if got[name].HasValue != want {
					t.Errorf("argument %s: expected HasValue %v, got %v", name, want, got[name].HasValue)
				}
			}
		})
	}
}

func TestArgumentValueAccessors(t *testing.T) {
	_, err := ParseArguments([]string{"name", "2", "yes", "x", "y"}, []Argument{
		{Name: "s", Type: ArgumentString},
		{Name: "i", Type: ArgumentInt},
		{Name: "b", Type: ArgumentBool},
		{Name: "l", Type: ArgumentList},
	})
	if err == nil {
		t.Fatal("expected error for unparseable bool 'yes'")
	}

	got, err := ParseArguments([]string{"name", "2", "true", "x", "y"}, []Argument{
		{Name: "s", Type: ArgumentString},
		{Name: "i", Type: ArgumentInt},
		{Name: "b", Type: ArgumentBool},
		{Name: "l", Type: ArgumentList},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got["s"].StringValue() != "name" {
		t.Errorf("StringValue: got %q", got["s"].StringValue())
	}
	if got["i"].IntValue() != 2 {
		t.Errorf("IntValue: got %d", got["i"].IntValue())
	}
	if !got["b"].BoolValue() {
		t.Errorf("BoolValue: got false")
	}
	if !reflect.DeepEqual(got["l"].ListValue(), []string{"x", "y"}) {
		t.Errorf("ListValue: got %v", got["l"].ListValue())
	}
}

func TestArgumentValueAccessorPanicsOnWrongType(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	Argument{Name: "s", Type: ArgumentString, Value: "x"}.IntValue()
}

func TestArgumentAsString(t *testing.T) {
	got := ArgumentAsString([]Argument{
		{Name: "name", Type: ArgumentString},
		{Name: "speed", Type: ArgumentString, Optional: true},
		{Name: "rest", Type: ArgumentList, Optional: true},
	})
	want := "<name> [speed] [rest...]"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestArgumentsString(t *testing.T) {
	got := ArgumentsString([]Argument{
		{Name: "name", Type: ArgumentString, Description: "the name"},
		{Name: "count", Type: ArgumentInt, Optional: true, Description: "how many"},
	})
	for _, want := range []string{"<name> string", "the name", "[count] int", "how many"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in %q", want, got)
		}
	}
}
