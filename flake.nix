{
  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    nixpkgs,
    flake-utils,
    ...
  }:
    flake-utils.lib.eachDefaultSystem (system: let
      pkgs = import nixpkgs {inherit system;};
    in {
      devShells.default =
        (pkgs.buildFHSEnv {
          name = "ical-bots";
          targetPkgs = pkgs:
            with pkgs; [
              bash
              gnumake
              buildPackages.stdenv.cc
              go
              # Needed for proto
              buf

              podman
              podman-compose
              # newuidmap
              su
            ];
        })
        .env;
    });
}
