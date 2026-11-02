# gmid

A Go desktop rewrite of [Orfeo](https://github.com/Squarebow/Orfeo): falling-note
piano roll, playable keyboard, chord and scale explorers, MIDI editing, and a
three-engine audio stack.

The reference Electron/React implementation lives in `Orfeo/` (read-only, not
part of this module).

## Status

Working desktop build: MIDI/MusicXML/KAR import, realtime playback (meltysynth +
malgo), falling-note roll, playable keyboard, mixer, focus mode, metronome and
count-in, hand assignment, step-through, tap tempo, chord/scale explorers,
playback and note editors, hardware MIDI input, WAV/MP3 export and chord-chart
PDFs.

Known gaps: Guitar Pro and Capella import use the local `mscore` command when
available (not bundled); OS-level fullscreen and multi-window; per-channel VU
meters are shared with the master peak; Starmap is a static chord-star view.
See `.operator/plans/roadmap.md` for milestones.

## Build requirements

- Go 1.27+
- A C compiler (cgo) and, on Linux, `libasound2-dev`
- `make` (optional)

```sh
make assets   # copies demo files/fonts; fetches GeneralUser GS if missing
make build
go run ./cmd/gmid
```

Headless screenshot (no display needed):

```sh
go run ./cmd/gmid -png build/visual/shell.png -scene=shell
```

## Test and lint

```sh
go test ./...
go vet ./...
golangci-lint run
```

## Visual QA

Screenshots are reviewed by the local `Qwen3.8-27B-VISION` model; images never
enter the coding model's context. See `.operator/guides/visual-qa.md`.

## License

GPL-3.0-or-later. Third-party assets and libraries are recorded in
`THIRD_PARTY_LICENSES.md`.
