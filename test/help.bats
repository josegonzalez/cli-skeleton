#!/usr/bin/env bats

load test_helper

@test "(help) --help goes to stdout and exits 0" {
  run --separate-stderr "$HELLO" --help
  assert_status 0
  assert_contains "$output" "USAGE"
  assert_contains "$output" "COMMANDS"
  [[ -z "$stderr" ]]
}

@test "(help) command help goes to stdout with examples in a stable order" {
  run --separate-stderr "$HELLO" eat --help
  assert_status 0
  first="$output"
  assert_contains "$first" "EXAMPLES"
  [[ "${first%%Eats one lollipop slowly*}" == *"Eats one lollipop quickly"* ]]

  for _ in 1 2 3 4 5; do
    run --separate-stderr "$HELLO" eat --help
    [[ "$output" == "$first" ]]
  done
}

@test "(help) help <command> shows the command help" {
  run --separate-stderr "$HELLO" help eat
  assert_status 0
  assert_contains "$output" "hello-world eat [flags] [speed]"
}

@test "(help) no arguments shows concise help on stderr and exits 2" {
  run --separate-stderr "$HELLO"
  assert_status 2
  [[ -z "$output" ]]
  assert_contains "$stderr" "USAGE"
  assert_contains "$stderr" "Run 'hello-world --help' for more information."
}

@test "(help) a parent command shows its subcommands" {
  run --separate-stderr "$NESTED" lollipop
  assert_status 2
  assert_contains "$stderr" "SUBCOMMANDS"
  assert_contains "$stderr" "eat     Eat lollipops from the jar"

  run --separate-stderr "$NESTED" lollipop --help
  assert_status 0
  assert_contains "$output" "SUBCOMMANDS"
}

@test "(help) unknown commands suggest the closest match and exit 127" {
  run -127 --separate-stderr "$HELLO" eatt
  assert_status 127
  assert_contains "$stderr" 'Unknown command "eatt"'
  assert_contains "$stderr" 'Did you mean "eat"?'
  [[ -z "$output" ]]

  run -127 --separate-stderr "$NESTED" lollipop lsit
  assert_status 127
  assert_contains "$stderr" 'Did you mean "list"?'
  refute_contains "$stderr" '"ls"'
}

@test "(help) unknown flags suggest the closest match and exit 2" {
  run --separate-stderr "$HELLO" eat --cuont 2
  assert_status 2
  assert_contains "$stderr" "unknown flag: --cuont"
  assert_contains "$stderr" "Did you mean --count?"
  [[ -z "$output" ]]
}

@test "(help) usage errors exit 2 with a short error on stderr" {
  run --separate-stderr "$HELLO" eat --count x
  assert_status 2
  assert_contains "$stderr" "invalid argument"
  assert_contains "$stderr" "For additional help try 'hello-world eat --help'"
  refute_contains "$stderr" "USAGE"
  [[ -z "$output" ]]
}

@test "(help) missing arguments show concise help" {
  run --separate-stderr "$NESTED" lollipop eat
  assert_status 2
  assert_contains "$stderr" "This command requires 1 argument: <flavor>"
  assert_contains "$stderr" "nested lollipop eat [flags] <flavor>"
}

@test "(help) global flags work before the subcommand" {
  run --separate-stderr "$GLOBAL" --global --global-string value global
  assert_status 0
  assert_contains "$output" "Global bool value: true"
  assert_contains "$output" "Global string value: value"

  for flags in "--quiet --no-color" "-q" "-qd"; do
    # shellcheck disable=SC2086
    run --separate-stderr "$HELLO" $flags eat
    assert_status 0
    assert_contains "$output" "Eating 1 normal lollipop(s) normally"
  done
}

@test "(help) the examples command prints every example" {
  run --separate-stderr "$NESTED" examples
  assert_status 0
  assert_contains "$output" "LOLLIPOP EAT"
  assert_contains "$output" "nested lollipop list --json"
}

@test "(help) version prints the version" {
  run --separate-stderr "$HELLO" version
  assert_status 0
  [[ "$output" == "0.0.0-test" ]]

  run --separate-stderr "$HELLO" --version
  assert_status 0
  [[ "$output" == "0.0.0-test" ]]

  run --separate-stderr "$HELLO" version --json
  assert_status 0
  assert_contains "$output" '"version": "0.0.0-test"'
  assert_contains "$output" '"go_version": "go'
}

@test "(help) help and version start quickly" {
  if [[ -z "${EPOCHREALTIME:-}" ]]; then
    skip "EPOCHREALTIME requires bash 5"
  fi

  for args in "--help" "version"; do
    timings=()
    for _ in 1 2 3 4 5; do
      start="${EPOCHREALTIME/[.,]/}"
      # shellcheck disable=SC2086
      "$HELLO" $args >/dev/null
      end="${EPOCHREALTIME/[.,]/}"
      timings+=($((end - start)))
    done

    median="$(printf '%s\n' "${timings[@]}" | sort -n | sed -n 3p)"
    if [[ "$median" -ge 100000 ]]; then
      echo "hello-world $args took ${median}us, expected under 100ms" >&2
      return 1
    fi
  done
}
