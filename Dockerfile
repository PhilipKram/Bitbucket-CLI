FROM golang:1.24-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bb .

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /bb /bb
VOLUME /config
ENV XDG_CONFIG_HOME=/config
EXPOSE 8080 8817
ENTRYPOINT ["/bb", "mcp", "serve", "--transport", "http", "--host", "0.0.0.0"]
