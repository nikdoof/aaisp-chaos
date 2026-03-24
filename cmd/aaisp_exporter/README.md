# AAISP Exporter

A Prometheus exporter for information about [Andrews and Arnold](https://aa.net.uk) broadband lines.

## Metrics

All metrics carry a `line_id` label containing the AAISP line ID.

| Metric | Type | Description |
|--------|------|-------------|
| `aaisp_broadband_line_info` | Gauge | Always `1`. Carries `line_id`, `login`, and `postcode` labels for use in dashboard queries. |
| `aaisp_broadband_tx_rate` | Gauge | Maximum download rate in bits per second (AAISP transmit). |
| `aaisp_broadband_tx_rate_adjusted` | Gauge | Adjusted download rate in bits per second after any throttling applied by AAISP. |
| `aaisp_broadband_rx_rate` | Gauge | Maximum upload rate in bits per second (AAISP receive). |
| `aaisp_broadband_quota_remaining` | Gauge | Remaining quota in the current monthly period, in bytes. |
| `aaisp_broadband_quota_total` | Counter | Total monthly quota in bytes. |
| `aaisp_scrape_success` | Gauge | `1` if the last scrape of the AAISP API succeeded, `0` otherwise. |

### Quota metrics

`aaisp_broadband_quota_remaining` and `aaisp_broadband_quota_total` are only emitted for lines with a **quota-limited** service. Lines on an unlimited quota plan (where the API returns `quota_monthly` of `0`) do not produce these metrics, avoiding misleading zero values.

### Joining line metadata in queries

`aaisp_broadband_line_info` is an info metric that can be joined onto other metrics to enrich graphs with the login name or postcode:

```promql
aaisp_broadband_tx_rate
  * on(line_id) group_left(login, postcode)
  aaisp_broadband_line_info
```

## Configuration

The following environment variables must be set:

| Variable | Description |
|----------|-------------|
| `CHAOS_CONTROL_LOGIN` | Control pages login, e.g. `something@a` |
| `CHAOS_CONTROL_PASSWORD` | Control pages password |

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-listen` | `:8080` | Address and port to bind to |
| `-log.level` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `-log.output` | `json` | Log format: `json` or `console` |

## Running

```sh
export CHAOS_CONTROL_LOGIN=something@a
export CHAOS_CONTROL_PASSWORD=yourpassword
./aaisp_exporter
```

Or with Docker:

```sh
docker run -e CHAOS_CONTROL_LOGIN=something@a \
           -e CHAOS_CONTROL_PASSWORD=yourpassword \
           -p 8080:8080 \
           ghcr.io/nikdoof/aaisp-exporter:latest
```

### Nix

Run directly without installing:

```sh
nix run github:nikdoof/aaisp-chaos -- -listen :8080
```

Install into a profile:

```sh
nix profile install github:nikdoof/aaisp-chaos
```

#### NixOS module

Add the flake as an input and import the module:

```nix
# flake.nix
{
  inputs = {
    aaisp-chaos.url = "github:nikdoof/aaisp-chaos";
  };

  outputs = { nixpkgs, aaisp-chaos, ... }: {
    nixosConfigurations.myhost = nixpkgs.lib.nixosSystem {
      modules = [
        aaisp-chaos.nixosModules.default
        {
          services.aaisp-exporter = {
            enable = true;
            listenAddress = ":8080";
            # Path to a file containing:
            #   CHAOS_CONTROL_LOGIN=something@a
            #   CHAOS_CONTROL_PASSWORD=yourpassword
            environmentFile = "/run/secrets/aaisp-exporter";
          };
        }
      ];
    };
  };
}
```

The service runs as a dynamic unprivileged user with a hardened systemd unit. Credentials are read from `environmentFile` at runtime and never written to the Nix store.

## Grafana dashboard

An example Grafana dashboard is provided at [`dashboards/aaisp-broadband.json`](../../dashboards/aaisp-broadband.json). Import it via **Dashboards → Import → Upload JSON file**.

The dashboard includes:

- Current download and upload rates
- Scrape status indicator
- Quota remaining (bytes and percentage) — only shown for quota-limited lines
- Download and upload rate history with max vs adjusted rate comparison
- Quota remaining over time
