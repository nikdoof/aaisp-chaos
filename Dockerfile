FROM golang:1.18.4-alpine3.16 as build
WORKDIR /build
COPY . .
RUN go get -d -v .
RUN go build -v ./cmd/aaisp_exporter

FROM alpine:3.16.0
WORKDIR /service
COPY --from=build /build/aaisp_exporter .
ENTRYPOINT ["./aaisp_exporter"]