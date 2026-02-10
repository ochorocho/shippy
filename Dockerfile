FROM debian:trixie-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        git \
        openssh-client \
        rsync \
        ca-certificates && \
    rm -rf /var/lib/apt/lists/*

ARG TARGETARCH
COPY dist/shippy-linux-${TARGETARCH} /usr/local/bin/shippy
RUN chmod +x /usr/local/bin/shippy

ENTRYPOINT ["shippy"]
