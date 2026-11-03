GO ?= go
SCENES ?= shell roll library mixer settings chord-explorer scale-explorer starmap note-editor presentation keys-floating export

# GeneralUser GS is not committed (31 MB). The URL is pinned to a commit so a
# release build always gets the same bank.
SOUNDFONT_URL ?= https://raw.githubusercontent.com/mrbumpy409/GeneralUser-GS/684543d5e5efaef08d02be50dcda8d552478fa60/GeneralUser-GS.sf2
SOUNDFONT_SHA256 ?= 9575028c7a1f589f5770fccc8cff2734566af40cd26ed836944e9a5152688cfe

.PHONY: all build run test lint fmt assets visual clean

all: build

build:
	$(GO) build -trimpath -o bin/gmid ./cmd/gmid

run: build
	./bin/gmid

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...
	golangci-lint run

fmt:
	$(GO) fmt ./...
	gofmt -w $$(find . -name '*.go' -not -path './Orfeo/*')

# Fonts and demo MIDIs are committed. The soundfont is fetched once and kept
# out of git (31 MB).
assets:
	@mkdir -p assets/fonts assets/demo assets/soundfonts build/visual
	@if [ ! -f assets/soundfonts/GeneralUser-GS.sf2 ]; then \
		echo "Fetching GeneralUser GS..."; \
		curl -fL --retry 3 -o assets/soundfonts/GeneralUser-GS.sf2 "$(SOUNDFONT_URL)" || \
			{ rm -f assets/soundfonts/GeneralUser-GS.sf2; \
			  echo "download failed: put GeneralUser-GS.sf2 in assets/soundfonts/"; exit 1; }; \
		got=$$(shasum -a 256 assets/soundfonts/GeneralUser-GS.sf2 2>/dev/null | awk '{print $$1}'); \
		[ -n "$$got" ] || got=$$(sha256sum assets/soundfonts/GeneralUser-GS.sf2 | awk '{print $$1}'); \
		if [ "$$got" != "$(SOUNDFONT_SHA256)" ]; then \
			rm -f assets/soundfonts/GeneralUser-GS.sf2; \
			echo "GeneralUser GS checksum mismatch: got $$got"; exit 1; \
		fi; \
	fi

visual: build
	@mkdir -p build/visual
	@for s in $(SCENES); do \
		./bin/gmid -png build/visual/$$s.png -scene=$$s >/dev/null 2>&1 || exit 1; \
		$(GO) run ./tools/visualqa -image build/visual/$$s.png -scene $$s || exit 1; \
	done

clean:
	rm -rf bin build
