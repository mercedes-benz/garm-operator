# SPDX-License-Identifier: MIT
FROM gcr.io/distroless/static:nonroot
LABEL org.opencontainers.image.source https://github.com/mercedes-benz/garm-operator
COPY garm-operator /manager
USER 65532:65532

ENTRYPOINT ["/manager"]