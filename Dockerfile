# syntax=docker/dockerfile:1

# Builds the gosnake binary and runs the multiplayer SSH server.
#   docker run -p 2222:2222 -v gosnake-data:/data ghcr.io/hilthontt/gosnake

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
      -ldflags "-s -w -X github.com/HilthonTT/gosnake/internal/telemetry.Version=${VERSION}" \
      -o /out/gosnake ./cmd/gosnake  && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/gosnake /usr/local/bin/gosnake
COPY --from=build --chown=65532:65532 /out/data /data
# The SSH host key lives in /data so it survives container restarts.
VOLUME /data
ENV GOSNAKE_KEY=/data/gosnake_server \
    GOSNAKE_PORT=2222
EXPOSE 2222
ENTRYPOINT ["gosnake", "serve"]
