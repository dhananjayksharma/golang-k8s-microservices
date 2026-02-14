# build
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/message-service ./cmd/server

# run
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /out/message-service /message-service
EXPOSE 8111
USER nonroot:nonroot
ENTRYPOINT ["/message-service"]
