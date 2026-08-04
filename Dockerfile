# Build the manager binary
FROM golang:1.26 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Private Go module config
ENV GIT_TERMINAL_PROMPT=1
ENV GOPROXY=direct
#RUN echo "machine github.com" > ~/.netrc
#RUN echo "login rforberger" >> ~/.netrc
#RUN --mount=type=secret,id=gh_token echo "password $(cat /run/secrets/gh_token)" >> ~/.netrc
#RUN --mount=type=secret,id=gh_token git config --global url."https://rforberger:$(cat /run/secrets/gh_token)@github.com/".insteadOf "https://github.com/"
RUN git config --global url."git@github.com:".insteadOf "https://github.com/"
ENV GOPRIVATE="github.com/rforberger/* gopkg.in/*"

# SSH host key configuration
RUN mkdir ~/.ssh
RUN echo "github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl" >> ~/.ssh/known_hosts

# SSH key config
#RUN cat $SSH_KEY > ~/.ssh/id_ed25519
#RUN cat $SSH_PUBLIC_KEY > ~/.ssh/id_ed25519.pub
COPY id_ed25519_demoapp /root/.ssh/id_ed25519
COPY id_ed25519_demoapp.pub /root/.ssh/id_ed25519.pub
RUN chmod 0600  /root/.ssh/id_ed25519


# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the Go source (relies on .dockerignore to filter)
COPY . .

# Build
# the GOARCH has no default value to allow the binary to be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
