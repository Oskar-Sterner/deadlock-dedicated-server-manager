.PHONY: build install clean

build:
	go build -o ddsm ./cmd/ddsm

install: build
	sudo install -m 0755 ddsm /usr/local/bin/ddsm

clean:
	rm -f ddsm
