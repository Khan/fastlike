.PHONY: build clean test

testdata/bin/main.wasm:
	cd testdata; \
	fastly compute build

build: testdata/bin/main.wasm

test:
	gotestsum ./... -race

clean:
	rm -rf testdata/bin/main.wasm
