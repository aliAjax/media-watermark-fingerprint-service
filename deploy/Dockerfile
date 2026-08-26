FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/media-server ./cmd/server
FROM alpine:3.20
RUN addgroup -S media && adduser -S -G media media
COPY --from=build /out/media-server /usr/local/bin/media-server
COPY configs /app/configs
USER media
WORKDIR /app
EXPOSE 8090 9090
ENTRYPOINT ["/usr/local/bin/media-server", "-config", "/app/configs/config.yaml"]
