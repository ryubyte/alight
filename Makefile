.PHONY: build run test clean install

build:
	go build -o aglight .

run: build
	./aglight

test:
	go test ./... -v

clean:
	rm -f aglight

install: build
	cp aglight /usr/local/bin/
