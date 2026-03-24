FROM golang:1.25-alpine3.21 AS build
ARG VERSION=dev
WORKDIR /build
COPY . .
RUN go mod download
RUN go build -ldflags "-X main.version=${VERSION}" -v ./cmd/aaisp_exporter

FROM alpine:3.23
WORKDIR /service
COPY --from=build /build/aaisp_exporter .
ENTRYPOINT ["./aaisp_exporter"]
