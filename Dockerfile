FROM alpine:3.16

# Dependencies
RUN apk add --no-cache ca-certificates

# Add pre-built application
COPY lil-dumpster /app
ENTRYPOINT [ "/app" ]

LABEL org.opencontainers.image.source https://github.com/team-dumpster-fire/lil-dumpster
