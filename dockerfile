# syntax=docker/dockerfile:1

FROM golang:1.26 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
RUN go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/torus \
    ./cmd/torus

FROM gcr.io/distroless/static-debian12

COPY --from=builder /out/torus /usr/local/bin/torus

ENTRYPOINT ["/usr/local/bin/torus"]
