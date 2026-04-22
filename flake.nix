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

        bkt = pkgs.buildGoModule {
          pname = "bkt";
          version = "dev";
          src = ./.;
          vendorHash = "sha256-Avpx630ORTnQ9FX+Umdv6tJ1g28i0rmW0eQuC3103Sk=";

          subPackages = [ "cmd/bkt" ];

          ldflags = [ "-s" "-w" ];

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
