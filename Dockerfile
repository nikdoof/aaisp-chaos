FROM golang:1.17.4-alpine3.15 as build
WORKDIR /build
COPY . .
RUN go get -d -v .
RUN go build -v ./cmd/aaisp_exporter

FROM alpine:3.15.4
WORKDIR /service
COPY --from=build /build/aaisp_exporter .
ENTRYPOINT ["./aaisp_exporter"]%