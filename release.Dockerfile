FROM gcr.io/distroless/static:nonroot
COPY garm-operator /manager
USER 65532:65532

ENTRYPOINT ["/manager"]