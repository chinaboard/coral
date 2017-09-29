all: binary ## Build all binaries

binary: 
	# go run chinaip_gen.go
	go build -ldflags="-s -w" -o bin/coral
