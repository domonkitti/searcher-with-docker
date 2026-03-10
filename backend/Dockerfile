# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/search-api ./cmd/server

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /out/search-api ./search-api
COPY data ./data
COPY deploy/entrypoint.sh ./entrypoint.sh
RUN chmod +x /app/entrypoint.sh
ENV ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]
