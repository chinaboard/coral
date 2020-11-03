all: data binary

binary:
	go build -o bin/coral cli/main.go

data:
	HTTP_PROXY=http://127.0.0.1:5438 go run utils/data/chinaip_gen.go

tidy:
	HTTP_PROXY=http://127.0.0.1:5438 go mod tidy