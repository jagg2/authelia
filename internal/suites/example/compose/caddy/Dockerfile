FROM caddy:2.8.4-builder@sha256:150c52a85b5d63f5c5892c20725df06669e8abf749584bfc24d3fa92f28c3ab4 AS builder

RUN xcaddy build fix-empty-copy-headers

FROM caddy:2.8.4@sha256:1841e7c656154710f2fec273e12d4e517eeea5bd7a6c75e01dd26b88aaba9646

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
