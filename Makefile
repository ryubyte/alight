.PHONY: build run test clean install

build:
	swift build -c release

run:
	swift run AgLight

test:
	swift run AgLightTestRunner

clean:
	rm -rf .build

install: build
	cp .build/release/AgLight /usr/local/bin/aglight
