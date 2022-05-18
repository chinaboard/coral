all: data binary

amd64:
	GOOS=linux GOARCH=amd64 go build -o bin/coral-amd64 cli/main.go
binary:
	go build -o bin/coral cli/main.go

data:
	go run utils/data/chinaip_gen.go

tidy:
	go mod tidy
