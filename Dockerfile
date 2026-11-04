FROM debian:trixie-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends openssh-client \
    && rm -rf /var/lib/apt/lists/*

ARG TARGETARCH
COPY dist/shippy-linux-${TARGETARCH} /usr/local/bin/shippy
RUN chmod +x /usr/local/bin/shippy

# To disable the entrypoint e.g. in GitLab CI use 'entrypoint: [""]'
# on the "image:" level in .gitlab-ci.yml so you can run regular commands
# inside the container.
ENTRYPOINT ["shippy"]
