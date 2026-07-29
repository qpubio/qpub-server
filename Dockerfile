# QPub Server
FROM golang:1.26-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/qpub-server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/qpub-server /app/qpub-server
USER nonroot:nonroot
EXPOSE 8081 8091 8111 8131
ENTRYPOINT ["/app/qpub-server"]
