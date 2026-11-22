#!/usr/bin/env bats

load test_helper

@test "(commands) confirmation without a terminal needs --force" {
  run --separate-stderr "$NESTED" lollipop eat cherry
  assert_status 2
  assert_contains "$stderr" "Confirmation required, but stdin is not a terminal."
  assert_contains "$stderr" "Pass --force to continue without confirming."
  [[ -z "$output" ]]

  run --separate-stderr "$NESTED" lollipop eat cherry --no-input
  assert_status 2
  assert_contains "$stderr" "--no-input was passed"

  run --separate-stderr "$NESTED" lollipop eat cherry --force
  assert_status 0

  run --separate-stderr "$NESTED" lollipop eat cherry -n
  assert_status 0
  [[ "$output" == "Would eat 1 cherry lollipop(s)" ]]
}

@test "(commands) arguments are validated before the command runs" {
  run --separate-stderr "$NESTED" lollipop eat banana --force
  assert_status 2
  assert_contains "$stderr" "Invalid value for argument flavor: must be one of cherry, grape, lemon, lime"
}

@test "(commands) deprecated commands warn and keep working" {
  run --separate-stderr "$NESTED" lollipop munch lemon --dry-run
  assert_status 0
  assert_contains "$stderr" 'Command "lollipop munch" is deprecated'
  [[ "$output" == "Would eat 1 lemon lollipop(s)" ]]

  run --separate-stderr "$NESTED" lollipop --help
  refute_contains "$output" "munch"
}

@test "(commands) aliases run their command" {
  run --separate-stderr "$NESTED" lollipop ls --plain
  assert_status 0
  [[ "${lines[0]}" == $'cherry\tred' ]]

  run --separate-stderr "$NESTED" lollipop list --help
  assert_contains "$output" "ALIASES"
  assert_contains "$output" "lollipop ls"
}

@test "(commands) config precedence is project over user, then env, then flags" {
  mkdir -p "$XDG_CONFIG_HOME/nested"
  echo "lollipop-eat:" >"$XDG_CONFIG_HOME/nested/config.yaml"
  echo "  count: 2" >>"$XDG_CONFIG_HOME/nested/config.yaml"

  run --separate-stderr "$NESTED" lollipop eat cherry -n
  [[ "$output" == "Would eat 2 cherry lollipop(s)" ]]

  mkdir -p project/sub
  echo "lollipop-eat:" >project/.nested.yaml
  echo "  count: 3" >>project/.nested.yaml
  cd project/sub
  run --separate-stderr "$NESTED" lollipop eat cherry -n
  [[ "$output" == "Would eat 3 cherry lollipop(s)" ]]

  export NESTED_COUNT=4
  run --separate-stderr "$NESTED" lollipop eat cherry -n
  [[ "$output" == "Would eat 4 cherry lollipop(s)" ]]

  run --separate-stderr "$NESTED" lollipop eat cherry -n --count 5
  [[ "$output" == "Would eat 5 cherry lollipop(s)" ]]

  run --separate-stderr "$NESTED" config --json
  assert_status 0
  assert_contains "$output" '"source": "env NESTED_COUNT"'
  assert_contains "$output" '.nested.yaml'
}

@test "(commands) quiet in the config file applies before the UI is built" {
  mkdir -p "$XDG_CONFIG_HOME/nested"
  echo "quiet: true" >"$XDG_CONFIG_HOME/nested/config.yaml"

  run --separate-stderr "$NESTED" lollipop eat cherry --force
  assert_status 0
  [[ -z "$stderr" ]]

  run --separate-stderr "$NESTED" lollipop eat cherry --force --quiet=false
  assert_contains "$stderr" "Next, run"
}

@test "(commands) invalid config files are reported" {
  mkdir -p "$XDG_CONFIG_HOME/nested"
  echo "quiet: [unclosed" >"$XDG_CONFIG_HOME/nested/config.yaml"

  run --separate-stderr "$NESTED" lollipop list
  assert_status 1
  assert_contains "$stderr" "Invalid config file $XDG_CONFIG_HOME/nested/config.yaml"
}

@test "(commands) docs are generated and installed" {
  run --separate-stderr "$NESTED" docs --output "$BATS_TEST_TMPDIR/docs"
  assert_status 0
  [[ -f "$BATS_TEST_TMPDIR/docs/index.md" ]]
  [[ -f "$BATS_TEST_TMPDIR/docs/lollipop-eat.md" ]]

  run --separate-stderr "$NESTED" docs --format man --install
  assert_status 2
  assert_contains "$stderr" "--force"
  [[ ! -e "$XDG_DATA_HOME/man/man1/nested.1" ]]

  run --separate-stderr "$NESTED" docs --format man --install --force
  assert_status 0
  [[ -f "$XDG_DATA_HOME/man/man1/nested.1" ]]
  [[ -f "$XDG_DATA_HOME/man/man1/nested-lollipop-eat.1" ]]
}

@test "(commands) completion installs only its own file" {
  touch "$HOME/.bashrc" "$HOME/.zshrc"
  before="$(ls -la "$HOME")"

  run --separate-stderr "$NESTED" completion bash --install
  assert_status 2
  [[ ! -e "$XDG_DATA_HOME/bash-completion/completions/nested" ]]

  run --separate-stderr "$NESTED" completion bash --install --force
  assert_status 0
  [[ -f "$XDG_DATA_HOME/bash-completion/completions/nested" ]]

  run --separate-stderr "$NESTED" completion zsh --install --force
  assert_status 0
  [[ -f "$XDG_DATA_HOME/zsh/site-functions/_nested" ]]

  [[ ! -s "$HOME/.bashrc" ]]
  [[ ! -s "$HOME/.zshrc" ]]

  run --separate-stderr "$NESTED" completion bash --uninstall
  assert_status 0
  [[ ! -e "$XDG_DATA_HOME/bash-completion/completions/nested" ]]
}

@test "(commands) completion requests are answered" {
  COMP_LINE="nested lollipop e" run --separate-stderr "$NESTED" nested e lollipop
  assert_status 0
  [[ "$output" == "eat" ]]
}
