FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/copyarr ./cmd/copyarr

FROM alpine:3.20
RUN apk add --no-cache ca-certificates rclone tzdata
COPY --from=build /out/copyarr /usr/local/bin/copyarr
VOLUME ["/data", "/config"]
EXPOSE 8686
ENTRYPOINT ["copyarr"]
CMD ["-config", "/config/config.json"]
