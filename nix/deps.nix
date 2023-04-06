{ pkgs
}:
with pkgs;
[
  # for running scripts
  bashInteractive
  # go building + testing
  go_1_20
  gcc
  # git plumbing
  git
  # for bud
  yarn
  # for mersk
  ruby
  bundix
]
