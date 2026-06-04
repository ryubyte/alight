.PHONY: build run test clean install

build:
	go build -o codex-bar .

run: build
	./codex-bar

test:
	go test ./... -v

clean:
	rm -f codex-bar

install: build
	cp codex-bar /usr/local/bin/
