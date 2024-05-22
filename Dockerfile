FROM docker.io/library/alpine:3.20 as runtime

RUN \
  apk add --update --no-cache \
    bash \
    curl \
    ca-certificates \
    tzdata

ENTRYPOINT ["provider-cloudscale"]
CMD ["operator"]
COPY provider-cloudscale /usr/bin/

USER 65536:0
