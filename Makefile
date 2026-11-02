GO ?= go
SCENES ?= shell roll library mixer settings chord-explorer scale-explorer note-editor tap-tempo presentation keys-floating

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
	@if [ ! -f assets/soundfonts/GeneralUser-GS.sf2 ] && [ -f Orfeo/public/GeneralUser-GS.sf2 ]; then \
		cp Orfeo/public/GeneralUser-GS.sf2 assets/soundfonts/; \
	elif [ ! -f assets/soundfonts/GeneralUser-GS.sf2 ]; then \
		echo "GeneralUser GS missing: copy it to assets/soundfonts/GeneralUser-GS.sf2"; \
	fi

visual: build
	@mkdir -p build/visual
	@for s in $(SCENES); do \
		./bin/gmid -png build/visual/$$s.png -scene=$$s >/dev/null 2>&1 || exit 1; \
		$(GO) run ./tools/visualqa -image build/visual/$$s.png -scene $$s || exit 1; \
	done

clean:
	rm -rf bin build
