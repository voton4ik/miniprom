FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/miniprom ./cmd/miniprom
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/exporter ./examples/exporter


FROM alpine:3.21 AS miniprom

WORKDIR /app

COPY --from=builder /out/miniprom .
COPY config.docker.json ./config.json

EXPOSE 9099

ENTRYPOINT ["./miniprom"]


FROM alpine:3.21 AS exporter

WORKDIR /app

COPY --from=builder /out/exporter .

EXPOSE 9100

ENTRYPOINT ["./exporter"]
