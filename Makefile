
GO_BUILD_ENV :=
GO_BUILD_FLAGS := -ldflags="-s -w"
MODULE_BINARY := bin/bluetooth-rescue

ifeq ($(VIAM_TARGET_OS), windows)
	GO_BUILD_ENV += GOOS=windows GOARCH=amd64
	GO_BUILD_FLAGS := -tags no_cgo	
	MODULE_BINARY = bin/bluetooth-rescue.exe
endif

$(MODULE_BINARY): Makefile go.mod *.go cmd/module/*.go 
	$(GO_BUILD_ENV) go build $(GO_BUILD_FLAGS) -o $@ ./cmd/module
	upx $@

lint:
	go vet ./...

update:
	go get go.viam.com/rdk@latest
	go mod tidy

test:
	go test ./...

module.tar.gz: meta.json $(MODULE_BINARY)
ifeq ($(VIAM_TARGET_OS), windows)
	jq '.entrypoint = "./bin/bluetooth-rescue.exe"' meta.json > temp.json && mv temp.json meta.json
endif
	tar czf $@ meta.json $(MODULE_BINARY)
ifeq ($(VIAM_TARGET_OS), windows)
	git checkout meta.json
endif

module: test module.tar.gz

all: test module.tar.gz

setup:
	go mod tidy

test-as-root:
	# need to be root to hit dmesg / kmsg
	go test . -o test-binary -run JustBuildDontTest
	sudo ./test-binary -test.v
