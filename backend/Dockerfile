FROM golang:1.26.8-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /server ./cmd/api
FROM alpine:3.22.1
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /server /app/server
USER app
EXPOSE 8000
ENTRYPOINT ["/app/server"]
