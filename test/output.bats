#!/usr/bin/env bats

load test_helper

@test "(output) piped output has no escape codes" {
  for binary in "$HELLO" "$HUMAN" "$ZEROLOG"; do
    run --separate-stderr "$binary" eat --count 2
    assert_status 0
    refute_contains "$output" "$ESC"
    refute_contains "$stderr" "$ESC"
  done

  run --separate-stderr "$HUMAN" eat --count x
  refute_contains "$stderr" "$ESC"
}

@test "(output) FORCE_COLOR colors piped output, and NO_COLOR wins" {
  export FORCE_COLOR=1
  run --separate-stderr "$HELLO" eat --count x
  assert_contains "$stderr" "$ESC["

  export NO_COLOR=1
  run --separate-stderr "$HELLO" eat --count x
  refute_contains "$stderr" "$ESC"

  unset NO_COLOR
  export NESTED_NO_COLOR=1
  run --separate-stderr "$NESTED" lollipop eat
  refute_contains "$stderr" "$ESC"

  unset NESTED_NO_COLOR
  run --separate-stderr "$NESTED" --no-color lollipop eat
  refute_contains "$stderr" "$ESC"
}

@test "(output) messages go to stderr and output to stdout" {
  run --separate-stderr "$HUMAN" eat
  assert_status 0
  assert_contains "$output" "Eating 1 normal lollipop(s) normally"

  run --separate-stderr "$NESTED" lollipop eat cherry --force
  assert_status 0
  assert_contains "$output" "Ate 1 cherry lollipop(s)"
  assert_contains "$stderr" "Next, run 'nested lollipop list'."
}

@test "(output) --quiet hides messages but keeps output" {
  run --separate-stderr "$NESTED" lollipop eat cherry --force --quiet
  assert_status 0
  assert_contains "$output" "Ate 1 cherry lollipop(s)"
  [[ -z "$stderr" ]]
}

@test "(output) tables support --plain and --json" {
  run --separate-stderr "$NESTED" lollipop list
  assert_status 0
  assert_contains "$output" "FLAVOR   COLOR"

  run --separate-stderr "$NESTED" lollipop list --plain
  assert_status 0
  [[ "${lines[0]}" == $'cherry\tred' ]]
  refute_contains "$output" "FLAVOR"

  run --separate-stderr "$NESTED" lollipop list --json --all
  assert_status 0
  assert_contains "$output" '"flavor": "lime"'
  assert_contains "$output" '"eaten": "true"'

  run --separate-stderr "$NESTED" lollipop list --json --plain
  assert_status 2
  assert_contains "$stderr" "--json and --plain cannot be used together"
}

@test "(output) --debug shows timestamps and value sources" {
  run --separate-stderr "$ZEROLOG" eat --debug
  assert_status 0
  [[ "$output" =~ [0-9]+:[0-9]+(AM|PM)\ INF ]]

  run --separate-stderr "$ZEROLOG" eat
  assert_status 0
  [[ "$output" == "Eating 1 normal lollipop(s) normally" ]]
}
