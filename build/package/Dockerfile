FROM golang:alpine as builder

ARG RELEASE_DATE=unknown
ARG VERSION=dev
ARG COMMIT=none

ENV CGO_ENABLED=0

RUN apk add git ca-certificates

WORKDIR /prometheus-multi-tenant-proxy

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY cmd cmd
COPY pkg pkg

RUN cd cmd/prometheus-multi-tenant-proxy && \
    go build -ldflags="-X 'main.version=${VERSION}' -X 'main.commit=${COMMIT}' -X 'main.date=${RELEASE_DATE}'"

FROM alpine

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /prometheus-multi-tenant-proxy/cmd/prometheus-multi-tenant-proxy/prometheus-multi-tenant-proxy /prometheus-multi-tenant-proxy

ENTRYPOINT [ "/prometheus-multi-tenant-proxy" ]
