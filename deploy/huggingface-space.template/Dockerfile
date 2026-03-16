FROM debian:bookworm-slim

RUN useradd --create-home --shell /usr/sbin/nologin appuser

WORKDIR /app
COPY pr-size-labeler /app/pr-size-labeler
RUN chmod 0755 /app/pr-size-labeler && chown appuser:appuser /app/pr-size-labeler

ENV LISTEN_ADDR=:7860
EXPOSE 7860

USER appuser
ENTRYPOINT ["/app/pr-size-labeler"]
