# Build stage
FROM golang:1.25 AS builder
WORKDIR /app

COPY go.mod ./
COPY main.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

# Run stage (small, secure-ish)
FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /app/server /server

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]
