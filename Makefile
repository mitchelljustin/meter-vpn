all: metervpn

metervpn:
	go build -o ./bin/metervpn ./cmd/metervpn

clean:
	rm bin/*

.PHONY: clean