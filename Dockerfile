FROM golang:1.18.3-alpine

# Build the app, dependencies first
RUN apk add --no-cache git

COPY go.mod go.sum /app/
WORKDIR /app
RUN go mod download

COPY . /app
ENV CGO_ENABLED=0
RUN go build -o main

# ---
FROM alpine:3.15 AS dist

# Dependencies
RUN apk add --no-cache ca-certificates

# Add pre-built application
COPY --from=0 /app/main /app
ENTRYPOINT [ "/app" ]

LABEL org.opencontainers.image.source https://github.com/team-dumpster-fire/lil-dumpster
