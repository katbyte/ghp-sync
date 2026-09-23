# syntax=docker/dockerfile:1

# no default: .go-version is the single source of truth and make/CI pass it in, so the builder can
# never lag go.mod's go directive (the official go images pin GOTOOLCHAIN=local, so an older
# builder fails the build rather than fetching a newer toolchain). build by hand with:
#   docker build --build-arg GO_VERSION=$(cat .go-version) .
ARG GO_VERSION

# the build stage runs on the builder's own architecture and cross-compiles for the target, so the
# multi-arch image builds in seconds rather than minutes under qemu emulation
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG GIT_COMMIT=docker

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -mod=vendor \
    -ldflags "-s -w -X github.com/katbyte/go-kt/version.Version=${VERSION} -X github.com/katbyte/go-kt/version.GitCommit=${GIT_COMMIT}" \
    -o /out/ghp-sync .

# dcron runs the sync on a schedule, gh backs the graphql queries, tzdata makes TZ work. upgrade first so a
# release rebuild picks up alpine security fixes the base image tag has not been rebuilt with yet
FROM alpine:3.24
RUN apk upgrade --no-cache && apk add --no-cache bash ca-certificates dcron github-cli tzdata

WORKDIR /app
COPY --from=build /out/ghp-sync /usr/bin/ghp-sync
COPY scripts/entry.sh scripts/run.sh /app/scripts/
RUN chmod +x /app/scripts/entry.sh /app/scripts/run.sh

CMD ["/app/scripts/entry.sh"]
