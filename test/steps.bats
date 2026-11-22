#!/usr/bin/env bats

load test_helper

@test "(steps) steps print a line as they start and end outside a terminal" {
  run --separate-stderr "$NESTED" lollipop savor --duration 0.3s
  assert_status 0
  assert_contains "$stderr" "-----> [1/2] Unwrapping the lollipop"
  assert_contains "$stderr" "       1 wrapper ("
  assert_contains "$stderr" "-----> [2/2] Checking the recipe"
  assert_contains "$stderr" "-----> Savoring"
  assert_contains "$stderr" "Finished in"
  refute_contains "$stderr" "$ESC"
  refute_contains "$stderr" "peeling the wrapper"
  assert_contains "$output" "Savored a lollipop"
}

@test "(steps) verbose mode streams every line" {
  NESTED_VERBOSE=1 run --separate-stderr "$NESTED" lollipop savor --duration 0.3s
  assert_status 0
  assert_contains "$stderr" "peeling the wrapper"
  assert_contains "$stderr" '"version": "0.0.0-test"'
}

@test "(steps) quiet mode prints nothing unless a step fails" {
  run --separate-stderr "$NESTED" lollipop savor --duration 0.3s --quiet
  assert_status 0
  [[ -z "$stderr" ]]

  run --separate-stderr "$NESTED" lollipop savor --duration 0.4s --quiet --drop
  assert_status 1
  assert_contains "$stderr" "Savoring failed after"
}

@test "(steps) failed steps show their output and the log" {
  run --separate-stderr "$NESTED" lollipop savor --duration 0.4s --drop
  assert_status 1
  assert_contains "$stderr" "Savoring failed after"
  assert_contains "$stderr" "the lollipop slipped"
  assert_contains "$stderr" " !     dropped the lollipop"
  assert_contains "$stderr" "Help: https://github.com/josegonzalez/cli-skeleton/issues"

  log="$(echo "$stderr" | sed -n 's/^ !     Full log: //p')"
  [[ "$log" == "$XDG_STATE_HOME/nested/logs/"* ]]
  [[ -f "$log" ]]
  assert_contains "$(cat "$log")" "peeling the wrapper"
  assert_contains "$(cat "$log")" "the lollipop slipped"
}

@test "(steps) a second run fails while the lock is held, or waits with --wait" {
  "$NESTED" lollipop savor --duration 2s --quiet &
  first=$!
  sleep 0.5

  run --separate-stderr "$NESTED" lollipop savor --duration 0.1s
  assert_status 1
  assert_contains "$stderr" 'Another nested process holds the "savor" lock'

  run --separate-stderr "$NESTED" lollipop savor --duration 0.1s --wait --quiet
  assert_status 0

  wait "$first"
}

@test "(steps) an interrupt stops the command and exits 130" {
  for signal in INT TERM; do
    "$NESTED" lollipop savor --duration 30s 2>"$BATS_TEST_TMPDIR/stderr" &
    pid=$!
    sleep 0.5
    kill -s "$signal" "$pid"

    status=0
    wait "$pid" || status=$?
    [[ "$status" -eq 130 ]]
    assert_contains "$(cat "$BATS_TEST_TMPDIR/stderr")" "Gracefully stopping... (press Ctrl+C again to force)"
    assert_contains "$(cat "$BATS_TEST_TMPDIR/stderr")" "Full log: "
  done
}
