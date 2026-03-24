{
  description = "Prometheus exporter for Andrews & Arnold broadband lines";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule rec {
          pname = "aaisp-exporter";
          version = "0.3.0";

          src = ./.;
          subPackages = [ "cmd/aaisp_exporter" ];
          ldflags = [ "-X main.version=${version}" ];

          # Run `nix build` with this set to lib.fakeHash to obtain the correct hash.
          vendorHash = "sha256-oeCSKwDKVwvYQ1fjXXTwQSXNl/upDE3WAAk680vqh3U=";

          meta = with pkgs.lib; {
            description = "Prometheus exporter for Andrews & Arnold broadband lines";
            homepage = "https://github.com/nikdoof/aaisp-chaos";
            license = licenses.mit;
            mainProgram = "aaisp_exporter";
            platforms = platforms.unix;
          };
        };

        apps.default = flake-utils.lib.mkApp {
          drv = self.packages.${system}.default;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [ go gopls gotools ];
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
                  "-listen" cfg.listenAddress
                  "-log.level" cfg.logLevel
                  "-log.output" cfg.logOutput
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
