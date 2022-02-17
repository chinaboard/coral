all: data binary

amd64:
	GOOS=linux GOARCH=amd64 go build -o bin/coral-amd64 cli/main.go
binary:
	go build -o bin/coral cli/main.go

data:
	HTTP_PROXY=http://127.0.0.1:5438 go run utils/data/chinaip_gen.go

tidy:
	HTTP_PROXY=http://127.0.0.1:5438 go mod tidy
