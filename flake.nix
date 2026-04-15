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

        version =
          if (self ? rev) then builtins.substring 0 7 self.rev
          else "dev";

        bkt = pkgs.buildGoModule {
          pname = "bkt";
          inherit version;
          src = ./.;
          vendorHash = "sha256-ZBYfb1B3OuD8nydEIy/tG1W03BjS1LUPepQvknUQO9Y=";

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
