FROM alpine:3.22.1

RUN apk add --no-cache ca-certificates \
    && wget -q "https://github.com/pressly/goose/releases/download/v3.26.0/goose_linux_x86_64" -O /usr/local/bin/goose \
    && chmod +x /usr/local/bin/goose

USER nobody
ENTRYPOINT ["goose", "-env", "none"]
