# SPDX-License-Identifier: MIT
FROM gcr.io/distroless/static:nonroot
LABEL org.opencontainers.image.source https://github.com/mercedes-benz/garm-operator
COPY garm-operator /manager
COPY tmp/3RD_PARTY_LICENSES.txt /3RD_PARTY_LICENSES.txt
USER 65532:65532

ENTRYPOINT ["/manager"]