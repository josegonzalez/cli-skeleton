package scaffold

import (
	"flag"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// exampleSpecs returns the scaffold.yaml of every example, keyed by the
// example name.
func exampleSpecs(t *testing.T) map[string]*Spec {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join("..", "..", "examples", "*", "scaffold.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no example specs found")
	}

	specs := map[string]*Spec{}
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		spec, err := ParseSpec(f)
		f.Close()
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		specs[filepath.Base(filepath.Dir(path))] = spec
	}
	return specs
}

func TestEveryExampleHasASpec(t *testing.T) {
	specs := exampleSpecs(t)
	dirs, err := os.ReadDir(filepath.Join("..", "..", "examples"))
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range dirs {
		if dir.IsDir() && specs[dir.Name()] == nil {
			t.Errorf("examples/%s has no scaffold.yaml", dir.Name())
		}
	}
}

func TestGenerateGolden(t *testing.T) {
	for name, spec := range exampleSpecs(t) {
		t.Run(name, func(t *testing.T) {
			files, err := Generate(spec, Options{SkeletonPath: "../../"})
			if err != nil {
				t.Fatal(err)
			}

			dir := filepath.Join("testdata", "golden", name)
			if *update {
				if err := os.RemoveAll(dir); err != nil {
					t.Fatal(err)
				}
				for _, file := range files {
					path := filepath.Join(dir, filepath.FromSlash(file.Path))
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, file.Contents, 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}

			var want []string
			err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}
				rel, err := filepath.Rel(dir, path)
				want = append(want, filepath.ToSlash(rel))
				return err
			})
			if err != nil {
				t.Fatalf("reading golden files: %v; run go test ./internal/scaffold -update", err)
			}
			sort.Strings(want)

			var got []string
			for _, file := range files {
				got = append(got, file.Path)
			}
			if strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Fatalf("generated files:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
			}

			for _, file := range files {
				golden, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(file.Path)))
				if err != nil {
					t.Fatal(err)
				}
				if string(golden) != string(file.Contents) {
					t.Errorf("%s differs from the golden file; run go test ./internal/scaffold -update\ngot:\n%s", file.Path, file.Contents)
				}
				if strings.HasSuffix(file.Path, ".go") {
					if _, err := parser.ParseFile(token.NewFileSet(), file.Path, file.Contents, parser.AllErrors); err != nil {
						t.Errorf("%s does not parse: %v", file.Path, err)
					}
				}
			}
		})
	}
}

func TestGenerateWithoutSkeletonPath(t *testing.T) {
	files, err := Generate(&Spec{Name: "app", Module: "example.com/app", Commands: []Command{{Name: "run", Synopsis: "Run it"}}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{}
	for _, file := range files {
		contents[file.Path] = string(file.Contents)
	}

	if got := contents["go.mod"]; got != "module example.com/app\n" {
		t.Errorf("go.mod = %q", got)
	}
	if !strings.Contains(contents["main.go"], `"example.com/app/commands"`) {
		t.Errorf("main.go does not import the commands package:\n%s", contents["main.go"])
	}
	if !strings.Contains(contents["main.go"], "Commands))") {
		t.Errorf("main.go should pass no options:\n%s", contents["main.go"])
	}
	if _, ok := contents["commands/run.go"]; !ok {
		t.Errorf("commands/run.go was not generated")
	}
	if _, ok := contents["commands/global_flags.go"]; ok {
		t.Errorf("commands/global_flags.go should only be generated with global flags")
	}
}

func TestGenerateRejectsInvalidSpecs(t *testing.T) {
	if _, err := Generate(&Spec{Name: "app"}, Options{}); err == nil {
		t.Fatal("expected an error for a spec without commands")
	}
}

func TestGenerateFlagTypes(t *testing.T) {
	spec := &Spec{Name: "app", Commands: []Command{{
		Name:     "run",
		Synopsis: "Run it",
		Flags: []Flag{
			{Name: "wait", Type: "duration", Default: "90s", Usage: "how long to wait"},
			{Name: "ratio", Type: "float", Default: "0.5", Usage: "the ratio"},
			{Name: "type", Usage: "the type"},
			{Name: "verbose-output", Type: "bool", Default: "true", Usage: "more output"},
		},
	}}}
	files, err := Generate(spec, Options{})
	if err != nil {
		t.Fatal(err)
	}
	var run string
	for _, file := range files {
		if file.Path == "commands/run.go" {
			run = string(file.Contents)
		}
	}

	for _, want := range []string{
		`"time"`,
		`f.DurationVar(&c.wait, "wait", 90*time.Second, "how long to wait")`,
		`f.Float64Var(&c.ratio, "ratio", 0.5, "the ratio")`,
		`f.StringVar(&c.typeValue, "type", "", "the type")`,
		`f.BoolVar(&c.verboseOutput, "verbose-output", true, "more output")`,
		`"--verbose-output": complete.PredictNothing`,
		`"--type":           complete.PredictAnything`,
	} {
		if !strings.Contains(run, want) {
			t.Errorf("commands/run.go does not contain %s:\n%s", want, run)
		}
	}
}
