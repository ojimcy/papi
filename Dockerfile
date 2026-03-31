FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o papi ./cmd/server/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/papi /usr/local/bin/papi
COPY --from=builder /app/migrations /migrations

EXPOSE 8080
CMD ["papi"]
