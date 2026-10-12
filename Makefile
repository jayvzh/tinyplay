# Go desktop build. The phone frontend lives in ./web and is embedded directly
# by the Go module; no generated frontend copy is needed.

VERSION ?= 0.0.0-dev

## ARCH: macOS build target, arm64 (Apple Silicon, default) or x86_64 (Intel).
## Ignored by build-core-win, which is always amd64.
ARCH ?= arm64
ifeq ($(ARCH),x86_64)
GOARCH_MAC := amd64
else
GOARCH_MAC := arm64
endif

.PHONY: sync tidy run build-core-mac build-core-win build-setup-win build-app-mac clean

## sync: compatibility no-op kept for older docs/scripts
sync:
	@true

tidy:
	go mod tidy

## run: run the headless core locally (Linux/macOS dev)
run:
	go run ./cmd/tvremote

## build-core-mac: the macOS core binary for ARCH (arm64 or x86_64; the native
## shell app bundles this)
build-core-mac:
	MACOSX_DEPLOYMENT_TARGET=12.0 CGO_CFLAGS=-mmacosx-version-min=12.0 CGO_LDFLAGS=-mmacosx-version-min=12.0 \
		GOOS=darwin GOARCH=$(GOARCH_MAC) go build -ldflags "-X main.version=$(VERSION)" -o build/tvremote-core-darwin-$(GOARCH_MAC) ./cmd/tvremote

## build-core-win: the Windows exe (tray + WebView2 shell, no console window)
build-core-win:
	GOOS=windows GOARCH=amd64 go build -ldflags "-H windowsgui -X main.version=$(VERSION)" -o build/TinyPlay.exe ./cmd/tvremote

## INNOSETUP_DIR: folder holding ISCC.exe. It is not on PATH by default, so the
## compiler has to be pointed at. Leave it empty to pick up an ISCC.exe that
## already is on PATH (that is what CI does).
INNOSETUP_DIR ?= D:/Code/tools/innosetup
ISCC ?= $(if $(INNOSETUP_DIR),$(INNOSETUP_DIR)/ISCC.exe,ISCC.exe)

## build-setup-win: the double-click Windows installer, dist/TinyPlay-Setup-x64.exe.
## Mirrors the windows leg of .github/workflows/release.yml, except that mpv is
## staged only when dist/mpv is missing — an already downloaded player is reused
## rather than re-fetched on every build.
build-setup-win:
	mkdir -p dist
	[ -d dist/mpv ] || bash ./windows/bundle-mpv.sh dist/mpv
	GOOS=windows GOARCH=amd64 go build -ldflags "-H windowsgui -X main.version=$(VERSION)" -o dist/TinyPlay.exe ./cmd/tvremote
	cp THIRD_PARTY_NOTICES.md dist/THIRD_PARTY_NOTICES.md
	"$(ISCC)" /DMyAppVersion=$(VERSION) windows/TinyPlay.iss

## build-app-mac: assemble the native .app around the core for ARCH (mpv via
## PATH unless MPV_DIR is set). CI sets MPV_DIR to a self-contained mpv;
## locally you can omit it and the app falls back to `mpv` on PATH.
build-app-mac: build-core-mac
	ARCH=$(ARCH) ./macos/build-app.sh

clean:
	rm -rf build dist mpvstage mpv_extract
