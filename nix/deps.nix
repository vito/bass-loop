{ pkgs
}:
with pkgs;
[
  # for running scripts
  bashInteractive
  # go building + testing
  go_1_18
  gcc
  # git plumbing
  git
  # for bud
  yarn
]
