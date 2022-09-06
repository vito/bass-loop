{ pkgs
}:
with pkgs;
[
  # for running scripts
  bashInteractive
  # go building + testing
  go_1_19
  gcc
  # git plumbing
  git
  # for bud
  yarn
]
