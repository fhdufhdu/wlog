FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/wlog ./cmd/wlog

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/* && useradd --create-home --uid 10001 wlog
WORKDIR /app
COPY --from=build --chown=wlog:wlog /out/wlog /app/wlog
COPY --chown=wlog:wlog styles.css admin.js mermaid.js math.js theme.js /app/
COPY --chown=wlog:wlog assets /app/assets
COPY --chown=wlog:wlog templates /app/templates
COPY --chown=wlog:wlog migrations /app/migrations
RUN mkdir -p /app/uploads && chown wlog:wlog /app/uploads
USER wlog
ENV BIND_ADDR=0.0.0.0:3000 UPLOAD_DIR=/app/uploads
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 CMD curl --fail --silent http://127.0.0.1:3000/health/live || exit 1
ENTRYPOINT ["/app/wlog"]
