FROM golang:1.23-alpine AS build
ARG VERSION=dev
ARG REVISION=unknown
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=$VERSION -X main.revision=$REVISION" -o /out/copyarr ./cmd/copyarr

FROM alpine:3.20
ARG VERSION=dev
ARG REVISION=unknown
LABEL org.opencontainers.image.title="Copyarr" \
      org.opencontainers.image.description="Persistent copy/move queue for Unraid and rclone-backed transfers" \
      org.opencontainers.image.source="https://github.com/swamp2k/copyarr" \
      org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.revision="$REVISION"
RUN apk add --no-cache ca-certificates rclone tzdata
COPY --from=build /out/copyarr /usr/local/bin/copyarr
VOLUME ["/data", "/config"]
EXPOSE 8686
ENTRYPOINT ["copyarr"]
CMD ["-config", "/config/config.json"]
