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

          vendorHash = "sha256-xgvmRPHWiUu/JaoUmYw674hqpdO6wvs4j/W2Cqxow1Q=";

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

          # Static analysis with golangci-lint.
          lint = pkgs.runCommand "check-golangci-lint"
            {
              nativeBuildInputs = [ pkgs.go pkgs.golangci-lint ];
              # golangci-lint needs a writable cache/home inside the sandbox.
            } ''
            export HOME=$TMPDIR
            export GOFLAGS=-mod=mod
            cp -r ${./.} ./src
            chmod -R u+w ./src
            cd ./src
            golangci-lint run ./... || exit 1
            touch $out
          '';
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
            gnumake
            jq
          ];

          shellHook = ''
            echo "terraform-provider-cachix dev shell"
            echo "  go:        $(go version | cut -d' ' -f3)"
            echo "  terraform: $(terraform version | head -n1)"
          '';
        };

        # `nix fmt` formats Nix files.
        formatter = pkgs.nixpkgs-fmt;
      }
    );
}
