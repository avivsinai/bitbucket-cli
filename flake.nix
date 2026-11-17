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
        version = "0.40.2";

        bkt = pkgs.buildGoModule {
          pname = "bkt";
          inherit version;
          src = ./.;
          vendorHash = "sha256-D/CCgj9ROPb8WIduvqmwjBGdEeMf+WD/tMeXlee1KfY=";

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
