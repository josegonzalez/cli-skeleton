#!/usr/bin/env bats

load test_helper

# setup moves HOME, so keep using the real Go caches instead of downloading
# every module again.
GO_PATH="$(go env GOPATH)"
GO_MODCACHE="$(go env GOMODCACHE)"
GO_BUILDCACHE="$(go env GOCACHE)"

# use_go_caches points go at the caches of the user running the tests.
use_go_caches() {
  export GOPATH="$GO_PATH" GOMODCACHE="$GO_MODCACHE" GOCACHE="$GO_BUILDCACHE"
}

# assert_generates_example generates the example named $1 from its spec,
# builds and tests it, and checks its docs match the docs of the example.
assert_generates_example() {
  local example="$1"
  local app="$BATS_TEST_TMPDIR/$example"
  use_go_caches

  run --separate-stderr "$SCAFFOLD" new "$app" --spec "$EXAMPLES/$example/scaffold.yaml" --skeleton-path "$REPO"
  assert_status 0
  assert_contains "$output" "Created $example"

  (cd "$app" && go build -ldflags "-X main.Version=0.0.0-test" -o "$example" . && go test ./...)

  "$app/$example" docs --output "$BATS_TEST_TMPDIR/generated-docs" >/dev/null 2>&1
  "$EXAMPLES/$example/$example" docs --output "$BATS_TEST_TMPDIR/example-docs" >/dev/null 2>&1
  diff -r "$BATS_TEST_TMPDIR/example-docs" "$BATS_TEST_TMPDIR/generated-docs"
}

@test "(scaffold) generates the global example" {
  assert_generates_example global
}

@test "(scaffold) generates the hello-world example" {
  assert_generates_example hello-world
}

@test "(scaffold) generates the human-readable-logging example" {
  assert_generates_example human-readable-logging
}

@test "(scaffold) generates the nested example" {
  assert_generates_example nested
}

@test "(scaffold) generates the nil example" {
  assert_generates_example nil
}

@test "(scaffold) generates the zerolog-logging example" {
  assert_generates_example zerolog-logging
}

@test "(scaffold) generated commands run" {
  use_go_caches
  run --separate-stderr "$SCAFFOLD" new app --spec "$EXAMPLES/nested/scaffold.yaml" --skeleton-path "$REPO"
  assert_status 0

  cd app
  go build -o nested .
  run --separate-stderr ./nested lollipop eat cherry --force
  assert_status 0

  run --separate-stderr ./nested lollipop ls --help
  assert_status 0
  assert_contains "$output" "List the lollipops in the jar"

  run --separate-stderr ./nested lollipop munch cherry --force
  assert_status 0
  assert_contains "$stderr" 'Command "lollipop munch" is deprecated'
}

@test "(scaffold) asks for a spec without a terminal" {
  run --separate-stderr "$SCAFFOLD" new app </dev/null
  assert_status 2
  assert_contains "$stderr" "Pass --spec with a YAML spec"
  [[ ! -e app ]]
}

@test "(scaffold) reads the spec from stdin" {
  run --separate-stderr "$SCAFFOLD" new app --spec - --no-tidy <"$EXAMPLES/nil/scaffold.yaml"
  assert_status 0
  [[ -f app/commands/nil.go ]]
  assert_contains "$stderr" "go mod tidy"
}

@test "(scaffold) dry runs write nothing" {
  run --separate-stderr "$SCAFFOLD" new app --spec "$EXAMPLES/nil/scaffold.yaml" --dry-run
  assert_status 0
  assert_contains "$output" "Would write app/main.go"
  assert_contains "$output" "Would run 'go mod tidy' in app"
  [[ ! -e app ]]
}

@test "(scaffold) existing files are kept without --force" {
  mkdir app
  echo "keep me" >app/main.go

  run --separate-stderr "$SCAFFOLD" new app --spec "$EXAMPLES/nil/scaffold.yaml" --no-tidy
  assert_status 1
  assert_contains "$stderr" "These files already exist"
  assert_contains "$stderr" "app/main.go"
  assert_contains "$stderr" "Pass --force"
  [[ "$(cat app/main.go)" == "keep me" ]]
  [[ ! -e app/commands ]]

  run --separate-stderr "$SCAFFOLD" new app --spec "$EXAMPLES/nil/scaffold.yaml" --no-tidy --force
  assert_status 0
  [[ "$(cat app/main.go)" != "keep me" ]]
}

@test "(scaffold) invalid specs are reported" {
  printf 'name: App\ncommands: []\n' >spec.yaml

  run --separate-stderr "$SCAFFOLD" new app --spec spec.yaml
  assert_status 2
  assert_contains "$stderr" "should be lowercase letters and digits"
  assert_contains "$stderr" "at least one command is required"
  [[ ! -e app ]]
}
