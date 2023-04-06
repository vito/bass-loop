{ lib
, pkgs
}:
pkgs.buildGo120Module rec {
  name = "bass-loop";
  src = ./.;

  vendorSha256 = lib.fileContents ./nix/vendorSha256.txt;

  # don't run tests here; they're too complicated
  doCheck = false;
}
