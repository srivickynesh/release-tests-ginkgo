# Dockerfile — Interop / downstream test image
# Layers fresh test source onto the pre-built CI tooling image (built by Dockerfile.CI).
# Used by interop and partner teams to run release tests without rebuilding all tooling.

ARG BASE_IMAGE=quay.io/openshift-pipeline/release-tests-ginkgo:latest
FROM ${BASE_IMAGE}

RUN mkdir -p /tmp/release-tests-ginkgo
WORKDIR /tmp/release-tests-ginkgo
COPY . .

# go build compiles packages; ginkgo build produces standalone .test binaries
# under each suite directory (e.g., tests/operator/operator.test) for direct execution.
RUN go build ./... && \
    ginkgo build ./tests/... && \
    echo "Compiled test binaries:" && find tests -name '*.test' | sort

RUN chgrp -R 0 /tmp && \
    chmod -R g=u /tmp

CMD ["/bin/bash"]
