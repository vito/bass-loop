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
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      rec {
        packages = {
          default = pkgs.callPackage ./default.nix { };
          # for passing to 'docker load'
          deps = pkgs.callPackage ./nix/depsImage.nix { };
          # for using as thunk images
          depsOci = pkgs.callPackage ./nix/convertToOci.nix {
            image = pkgs.callPackage ./nix/depsImage.nix { };
          };
        };

        defaultApp = {
          type = "app";
          program = "${packages.bass-loop}/bin/bass-loop";
        };

        devShells = {
          default = pkgs.mkShell {
            nativeBuildInputs = pkgs.callPackage ./nix/deps.nix { } ++ (with pkgs; [
              gopls
            ]);
          };
        };
      });
}
