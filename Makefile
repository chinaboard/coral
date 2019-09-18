all: data binary

binary:
	go build -o bin/coral

data:
	go run utils/data/chinaip_gen.go