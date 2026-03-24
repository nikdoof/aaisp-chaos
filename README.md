# Andrews and Arnold CHAOS API

The `chaos` package provides a Go client for the [Andrews and Arnold](https://aa.net.uk) CHAOS v2 API.

## Implemented

* Broadband info (`/broadband/info`)
* Broadband quota (`/broadband/quota`)

## Not implemented

Write operations (ordering, ceasing, adjusting services) and PPP kill are out of scope for this library's current use as a read-only Prometheus exporter backend.

## Exporter

See [cmd/aaisp_exporter](cmd/aaisp_exporter/README.md) for the Prometheus exporter built on this package.
