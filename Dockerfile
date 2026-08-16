FROM golang:1.24-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go run github.com/a-h/templ/cmd/templ@v0.3.960 generate
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/station ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata ffmpeg poppler-utils \
	&& adduser -D -H -u 10001 station
WORKDIR /app
COPY --from=build /out/station /app/station
USER station
EXPOSE 8080
ENTRYPOINT ["/app/station"]
