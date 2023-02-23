FROM golang:1.20-alpine as builder
RUN apk --no-cache add git

WORKDIR /build
COPY go.* ./
RUN go mod download

COPY . .
RUN go build -o /bin/api ./cmd/api

# --- Execution Stage

FROM alpine:latest
RUN apk --no-cache add ffmpeg

COPY --from=builder /bin/api /bin/

EXPOSE 80
ENTRYPOINT ["/bin/api"]
