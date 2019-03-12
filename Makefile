all: meterd

meterd:
	go build -o ./bin/meterd ./cmd/meterd