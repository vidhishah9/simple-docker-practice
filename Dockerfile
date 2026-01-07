# Build stage

#start from the official go image that already has the go compiler installed 
FROM golang:1.25 AS builder  

#sets working directory to /app inside the container
WORKDIR /app

#copies go.mod to the container at /app/go.mod
COPY go.mod ./

#copies main.go to the container at /app/main.go
COPY main.go ./

#execute this container on linux, compile main.go and put its output to a binary named server (which contains the app basically)
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

# Run stage (small, secure-ish)
FROM gcr.io/distroless/static-debian12
WORKDIR /

#copies binary from the builder stage to the run stage
COPY --from=builder /app/server /server 

EXPOSE 8080
USER nonroot:nonroot

#entrypoint that runs when you run the docker container
ENTRYPOINT ["/server"]
