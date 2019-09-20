all: data binary

binary:
	go build -o bin/coral cli/main.go

data:
	go run utils/data/chinaip_gen.go