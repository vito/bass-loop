{
  description = "a continuous Bass server";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      supportedSystems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-darwin"
        "x86_64-linux"
      ];
    in
    flake-utils.lib.eachSystem supportedSystems (system:
      with (nixpkgs.legacyPackages.${system});
      let
        env = bundlerEnv {
          name = "bass-loop-bundler-env";
          inherit ruby;
          gemfile = ./Gemfile;
          lockfile = ./Gemfile.lock;
          gemset = ./gemset.nix;
        };
      in
      rec {
        packages = {
          default = callPackage ./default.nix { };
        };

        devShells = {
          default = mkShell {
            nativeBuildInputs = callPackage ./nix/deps.nix { } ++ [
              gopls
              env
            ];
          };
        };
      });
}
