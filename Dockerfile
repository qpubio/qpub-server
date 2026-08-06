# QPub Server
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/qpub-server ./cmd/server

# alpine avoids gcr.io/distroless pulls (often 403 from local networks)
FROM alpine:latest
WORKDIR /app
COPY --from=builder /out/qpub-server /app/qpub-server
RUN adduser -D -u 65532 nonroot \
	&& mkdir -p /app/logs \
	&& chown -R nonroot:nonroot /app
USER nonroot
EXPOSE 8091 8111 8131
ENTRYPOINT ["/app/qpub-server"]
