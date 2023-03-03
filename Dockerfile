FROM golang:1.20-alpine as builder
RUN apk --no-cache add git make build-base

WORKDIR /build
COPY go.* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o /bin/api ./cmd/api

# --- Execution Stage

FROM alpine:latest
RUN apk --no-cache add ffmpeg

COPY --from=builder /bin/api /bin/

EXPOSE 80
ENTRYPOINT ["/bin/api"]
