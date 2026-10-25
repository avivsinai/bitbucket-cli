{
  description = "bkt – Bitbucket CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        # Kept in sync with the released version by scripts/release.sh.
        version = "dev";

        bkt = pkgs.buildGoModule {
          pname = "bkt";
          inherit version;
          src = ./.;
          vendorHash = "sha256-S8g9UK8rjTBj9C99LHv+WnH3tRu/nJjP0xNBRuGmrZQ=";

          subPackages = [ "cmd/bkt" ];

          ldflags = [
            "-s"
            "-w"
            "-X github.com/avivsinai/bitbucket-cli/internal/build.versionFromLdflags=${version}"
          ];

          meta = with pkgs.lib; {
            description = "Bitbucket CLI (gh-equivalent for Bitbucket Cloud and Data Center)";
            homepage = "https://github.com/avivsinai/bitbucket-cli";
            license = licenses.mit;
            mainProgram = "bkt";
          };
        };
      in
      {
        packages.default = bkt;
        packages.bkt = bkt;

        apps.default = {
          type = "app";
          program = "${bkt}/bin/bkt";
        };

        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go pkgs.gopls pkgs.gotools pkgs.golangci-lint ];
        };
      });
}
