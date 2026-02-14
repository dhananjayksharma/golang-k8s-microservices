FROM golang:1.25 as builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app ./cmd/api

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /app /app
EXPOSE 8914
USER nonroot:nonroot
ENTRYPOINT ["/app"]
