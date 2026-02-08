# --- Build stage ---
FROM golang:1.22-alpine AS builder

WORKDIR /src

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/dmud -v ./cmd/dmud

# --- Runtime stage ---
FROM alpine:3.18

RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/dmud /usr/local/bin/dmud
COPY resources/ /app/resources/

WORKDIR /app

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["dmud"]
