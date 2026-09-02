FROM ubuntu:24.04

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home --uid 10001 wlog

WORKDIR /app

COPY --chown=wlog:wlog target/release/wlog /app/wlog
COPY --chown=wlog:wlog styles.css admin.js mermaid.js theme.js /app/
COPY --chown=wlog:wlog assets /app/assets

RUN mkdir -p /app/uploads && chown wlog:wlog /app/uploads

USER wlog

ENV BIND_ADDR=0.0.0.0:3000 \
    UPLOAD_DIR=/app/uploads

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl --fail --silent http://127.0.0.1:3000/health/live || exit 1

ENTRYPOINT ["/app/wlog"]
