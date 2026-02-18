FROM alpine:3.19
WORKDIR /
COPY manager /manager

USER 65532:65532

ENTRYPOINT ["/manager"]
