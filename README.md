Simple go server setup docker

1. Create the go file

```
go mod init tiny-go-docker
go mod tidy
go run main.go (runs the go server)
```
2. Create the docker file, build the image, run the image
```
docker build -t my-go-app . 
docker run --rm -p 8080:8080 my-go-app 
```
