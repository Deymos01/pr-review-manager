FROM golang:1.25.4 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/bin/app ./cmd/app
RUN go build -o /app/bin/migrator ./cmd/migrator

FROM debian:bookworm-slim AS runtime

WORKDIR /app
COPY --from=builder /app/bin/app /app/bin/app
COPY --from=builder /app/bin/migrator /app/bin/migrator
COPY migrations ./migrations
COPY configs ./configs

EXPOSE 8080

ENV CONFIG_PATH=./configs/docker.yaml

CMD ["/app/bin/app"]