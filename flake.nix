{
  description = "Prometheus exporter for Andrews & Arnold broadband lines";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem
      (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          version = "0.3.0";
          vendorHash = "sha256-REEJCTpSx2Kt5J93n6vS3nnpl+StPx26nBCzOgV/PDw=";
        in
        {
          packages.default = pkgs.buildGoModule rec {
            pname = "aaisp-exporter";
            inherit version vendorHash;

            src = ./.;
            subPackages = [ "cmd/aaisp_exporter" ];
            ldflags = [ "-X main.version=${version}" ];

            meta = with pkgs.lib; {
              description = "Prometheus exporter for Andrews & Arnold broadband lines";
              homepage = "https://github.com/nikdoof/aaisp-chaos";
              license = licenses.mit;
              mainProgram = "aaisp_exporter";
              platforms = platforms.unix;
            };
          };

          checks = {
            # Run the full test suite across all packages
            tests = pkgs.buildGoModule {
              pname = "aaisp-exporter-tests";
              inherit version vendorHash;
              src = ./.;
              doCheck = true;
              installPhase = "touch $out";
            };

            # Run golangci-lint
            lint = pkgs.buildGoModule {
              pname = "aaisp-exporter-lint";
              inherit version vendorHash;
              src = ./.;
              nativeBuildInputs = [ pkgs.golangci-lint ];
              buildPhase = ''
                export GOLANGCI_LINT_CACHE=$TMPDIR
                golangci-lint run ./...
              '';
              installPhase = "touch $out";
              doCheck = false;
            };

            # Check Nix formatting
            fmt = pkgs.runCommand "nixpkgs-fmt-check" { } ''
              ${pkgs.nixpkgs-fmt}/bin/nixpkgs-fmt --check ${./flake.nix}
              touch $out
            '';
          };

          apps.default = flake-utils.lib.mkApp {
            drv = self.packages.${system}.default;
          };

          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              # Go toolchain
              go
              gopls
              gotools # goimports, godoc, etc.
              golangci-lint

              # Nix development
              nil # Nix language server
              nixpkgs-fmt # Nix formatter
            ];

            shellHook = ''
              echo "AAISP Chaos development shell"
              echo ""
              echo "Go commands:"
              echo "  go build ./cmd/aaisp_exporter  build the exporter"
              echo "  go test ./...                  run all tests"
              echo "  golangci-lint run              lint the codebase"
              echo ""
              echo "Nix commands:"
              echo "  nix build                      build the package"
              echo "  nix flake check                run all checks"
              echo "  nixpkgs-fmt flake.nix          format the flake"
              echo ""
            '';
          };
        }
      ) // {
      nixosModules.default = { config, lib, pkgs, ... }:
        let
          cfg = config.services.aaisp-exporter;
          pkg = self.packages.${pkgs.system}.default;
        in
        {
          options.services.aaisp-exporter = {
            enable = lib.mkEnableOption "AAISP Prometheus exporter";

            listenAddress = lib.mkOption {
              type = lib.types.str;
              default = ":8080";
              example = ":9090";
              description = "Address and port the exporter listens on.";
            };

            logLevel = lib.mkOption {
              type = lib.types.enum [ "debug" "info" "warn" "error" ];
              default = "info";
              description = "Log verbosity level.";
            };

            logOutput = lib.mkOption {
              type = lib.types.enum [ "json" "console" ];
              default = "json";
              description = "Log output format.";
            };

            environmentFile = lib.mkOption {
              type = lib.types.path;
              example = "/run/secrets/aaisp-exporter";
              description = ''
                Path to a file containing credentials as environment variables.
                The file must define:
                  CHAOS_CONTROL_LOGIN=something@a
                  CHAOS_CONTROL_PASSWORD=yourpassword

                This file should not be in the Nix store. Use a secrets
                manager (e.g. agenix, sops-nix) to provision it.
              '';
            };
          };

          config = lib.mkIf cfg.enable {
            systemd.services.aaisp-exporter = {
              description = "AAISP Prometheus exporter";
              wantedBy = [ "multi-user.target" ];
              after = [ "network.target" ];

              serviceConfig = {
                ExecStart = lib.escapeShellArgs [
                  "${pkg}/bin/aaisp_exporter"
                  "-listen"
                  cfg.listenAddress
                  "-log.level"
                  cfg.logLevel
                  "-log.output"
                  cfg.logOutput
                ];

                EnvironmentFile = cfg.environmentFile;

                # Run as a dynamic unprivileged user
                DynamicUser = true;
                Restart = "on-failure";
                RestartSec = "5s";

                # Hardening
                NoNewPrivileges = true;
                PrivateTmp = true;
                PrivateDevices = true;
                ProtectSystem = "strict";
                ProtectHome = true;
                ProtectKernelTunables = true;
                ProtectKernelModules = true;
                ProtectControlGroups = true;
                CapabilityBoundingSet = "";
                AmbientCapabilities = "";
                SystemCallArchitectures = "native";
                SystemCallFilter = "@system-service";
                LockPersonality = true;
                RestrictRealtime = true;
                RestrictNamespaces = true;
              };
            };
          };
        };
    };
}
