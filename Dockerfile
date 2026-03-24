FROM golang:1.22-alpine3.21 AS build
WORKDIR /build
COPY . .
RUN go mod download
RUN go build -v ./cmd/aaisp_exporter

FROM alpine:3.21
WORKDIR /service
COPY --from=build /build/aaisp_exporter .
ENTRYPOINT ["./aaisp_exporter"]
