#!/usr/bin/env bash
# shellcheck disable=SC2034,SC2154
# The variables defined here are used by the test files, and status, output
# and stderr are set by bats.

bats_require_minimum_version 1.5.0

REPO="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
EXAMPLES="$REPO/examples"
SCAFFOLD="$REPO/cmd/cli-skeleton/cli-skeleton"
GLOBAL="$EXAMPLES/global/global"
HELLO="$EXAMPLES/hello-world/hello-world"
HUMAN="$EXAMPLES/human-readable-logging/human-readable-logging"
NESTED="$EXAMPLES/nested/nested"
ZEROLOG="$EXAMPLES/zerolog-logging/zerolog-logging"
ESC=$'\033'

setup() {
  for binary in "$GLOBAL" "$HELLO" "$HUMAN" "$NESTED" "$ZEROLOG" "$SCAFFOLD"; do
    if [[ ! -x "$binary" ]]; then
      echo "missing $binary; run 'make build' first" >&2
      return 1
    fi
  done

  unset NO_COLOR FORCE_COLOR DEBUG PAGER LESS COMP_LINE
  unset NESTED_COUNT NESTED_DEBUG NESTED_VERBOSE NESTED_PAGER NESTED_DEBUG_LOG
  export TERM=xterm-256color
  export HOME="$BATS_TEST_TMPDIR/home"
  export XDG_CONFIG_HOME="$HOME/.config"
  export XDG_DATA_HOME="$HOME/.local/share"
  export XDG_STATE_HOME="$HOME/.local/state"
  export XDG_CACHE_HOME="$HOME/.cache"
  export XDG_RUNTIME_DIR="$BATS_TEST_TMPDIR/run"
  mkdir -p "$HOME" "$XDG_RUNTIME_DIR"
  cd "$BATS_TEST_TMPDIR" || return 1
}

# assert_contains fails unless $1 contains $2.
assert_contains() {
  if [[ "$1" != *"$2"* ]]; then
    echo "expected to find: $2" >&2
    echo "in: $1" >&2
    return 1
  fi
}

# refute_contains fails when $1 contains $2.
refute_contains() {
  if [[ "$1" == *"$2"* ]]; then
    echo "expected not to find: $2" >&2
    echo "in: $1" >&2
    return 1
  fi
}

# assert_status fails unless the last run exited with $1.
assert_status() {
  if [[ "$status" -ne "$1" ]]; then
    echo "expected exit status $1, got $status" >&2
    echo "stdout: $output" >&2
    echo "stderr: ${stderr:-}" >&2
    return 1
  fi
}
