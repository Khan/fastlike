.PHONY: build
.PHONY: clean

testdata/bin/main.wasm:
	cd testdata; \
	fastly compute build

build: testdata/bin/main.wasm

clean:
	rm -rf testdata/bin/main.wasm
