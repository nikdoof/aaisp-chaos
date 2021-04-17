FROM alpine

COPY cmd/aaisp_exporter/aaisp_exporter /usr/local/bin/aaisp_exporter

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/aaisp_exporter"]
