.PHONY: build-arm
build-arm: install-packr
	GOARCH=arm GOARM=7 packr build -ldflags="-s -w" ./cmd/cacophony-config-sync

.PHONY: install-packr
install-packr:
	go install github.com/gobuffalo/packr/packr@v1.30.1

.PHONY: build
build: install-packr
	packr build -ldflags="-s -w" ./cmd/cacophony-config-sync

.PHONY: release
release: install-packr
	curl -sL https://git.io/goreleaser | bash

.PHONY: clean
clean:
	packr clean
	rm cacophony-config-sync
