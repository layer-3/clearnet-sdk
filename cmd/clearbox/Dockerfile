FROM golang:1.27.2-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY cmd/clearbox ./cmd/clearbox
COPY internal/clearbox ./internal/clearbox
COPY pkg ./pkg
RUN CGO_ENABLED=1 go build -mod=readonly -trimpath -ldflags="-s -w" -o /clearbox ./cmd/clearbox

FROM debian:bookworm-slim
RUN install -d -o 10001 -g 10001 -m 0700 /data
COPY --from=build /clearbox /usr/local/bin/clearbox
USER 10001:10001
WORKDIR /data
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/clearbox"]
