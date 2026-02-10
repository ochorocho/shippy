FROM debian:trixie-slim

ARG TARGETARCH
COPY dist/shippy-linux-${TARGETARCH} /usr/local/bin/shippy
RUN chmod +x /usr/local/bin/shippy

# To disable the entrypoint e.g. in GitLab CI use 'entrypoint: [""]'
# on the "image:" level in .gitlab-ci.yml so you can run regular commands
# inside the container.
ENTRYPOINT ["shippy"]
