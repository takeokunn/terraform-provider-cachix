{
  description = "Terraform provider for managing Cachix binary caches";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { self
    , nixpkgs
    , flake-utils
    ,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          # Terraform is distributed under the BUSL license, which Nix treats as unfree.
          config.allowUnfree = true;
        };

        version = "1.1.0";

        terraform-provider-cachix = pkgs.buildGoModule {
          pname = "terraform-provider-cachix";
          inherit version;

          src = ./.;

          vendorHash = "sha256-NgdUaB1Src9B/AlJLkV5Uf1L2PEPcnDRQ3De4r5KXIc=";

          # Strip debug info and stamp the version, mirroring .goreleaser.yml.
          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
          ];

          env.CGO_ENABLED = 0;

          subPackages = [ "." ];

          # Run the unit test suite during the build. Acceptance tests
          # (TestAcc*) require TF_ACC and are skipped automatically.
          doCheck = true;

          meta = with pkgs.lib; {
            description = "Terraform provider for managing Cachix binary caches";
            homepage = "https://github.com/takeokunn/terraform-provider-cachix";
            license = licenses.mpl20;
            mainProgram = "terraform-provider-cachix";
          };
        };
      in
      {
        # `nix build` produces the provider binary.
        packages = {
          inherit terraform-provider-cachix;
          default = terraform-provider-cachix;
        };

        # `nix flake check` runs every check below.
        checks = {
          # Compiles all packages and runs unit tests.
          build = terraform-provider-cachix;

          # Fail if any Go source is not gofmt-formatted.
          gofmt = pkgs.runCommand "check-gofmt" { nativeBuildInputs = [ pkgs.go ]; } ''
            cd ${./.}
            unformatted="$(gofmt -l -s .)"
            if [ -n "$unformatted" ]; then
              echo "The following files are not gofmt-formatted:"
              echo "$unformatted"
              exit 1
            fi
            touch $out
          '';

          # Static analysis with golangci-lint. buildGoModule's goModules
          # output is a Go vendor directory; symlinking it as ./vendor lets the
          # Go toolchain resolve every import offline via its standard vendor
          # auto-detection (go >= 1.14), so this works inside the
          # network-isolated Nix sandbox on Linux CI.
          lint = pkgs.stdenv.mkDerivation {
            pname = "terraform-provider-cachix-lint";
            inherit version;
            src = ./.;
            nativeBuildInputs = [ pkgs.go pkgs.golangci-lint ];
            configurePhase = ''
              runHook preConfigure
              ln -s ${terraform-provider-cachix.goModules} vendor
              runHook postConfigure
            '';
            buildPhase = ''
              runHook preBuild
              export HOME=$TMPDIR
              export GOCACHE=$TMPDIR/go-cache
              export GOLANGCI_LINT_CACHE=$TMPDIR/golangci-lint-cache
              golangci-lint run ./...
              runHook postBuild
            '';
            installPhase = ''
              runHook preInstall
              touch $out
              runHook postInstall
            '';
          };
        };

        # `nix develop` drops into a shell with the full toolchain.
        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            # Go toolchain
            go
            gopls
            golangci-lint
            goreleaser

            # Terraform ecosystem
            terraform
            terraform-docs
            terraform-ls

            # Utilities
            jq
          ];

          shellHook = ''
            echo "terraform-provider-cachix dev shell"
            echo "  go:        $(go version | cut -d' ' -f3)"
            echo "  terraform: $(terraform version | head -n1)"
          '';
        };

        # Runnable entry points (replacing the Makefile).
        apps = {
          # `nix run .#docs` regenerates the registry documentation.
          docs = {
            type = "app";
            program = pkgs.lib.getExe (pkgs.writeShellApplication {
              name = "generate-docs";
              runtimeInputs = [ pkgs.go pkgs.terraform ];
              text = "go generate ./...";
            });
          };

          # `nix run .#testacc` runs the acceptance tests. Requires
          # CACHIX_AUTH_TOKEN and creates real resources against the Cachix API.
          testacc = {
            type = "app";
            program = pkgs.lib.getExe (pkgs.writeShellApplication {
              name = "testacc";
              runtimeInputs = [ pkgs.go ];
              text = ''
                TF_ACC=1 go test -v -timeout 120m ./...
              '';
            });
          };
        };

        # `nix fmt` formats Nix files.
        formatter = pkgs.nixpkgs-fmt;
      }
    );
}
