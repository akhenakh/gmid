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

DecentSampler instruments (`*.dspreset`, `*.dslibrary`) play through a pure-Go
sampler: key/velocity zones, round robins, loops, amp envelopes, low-pass
filters, CC filters and per-channel mix. Melodic tracks use the sampler; drums
keep the soundfont. Libraries are level-matched at load (with a per-instrument
gain in Settings → Audio), the demo piano lives in `assets/decent/`, and the
`gmid_decent` MCP tool loads, unloads and inspects instruments.

A loaded DecentSampler instrument replaces the melodic synth for every
non-drum track, so the GM/Studio/Native engine only affects drums until it is
unloaded. GM uses the bundled General User GS; Studio and Native load the SF2
set in Settings → Audio (the GM bank until one is chosen). Native differs by
its exclusive low-latency output, not by its bank.

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

## MCP (drive gmid from an LLM)

`gmid` starts a Model Context Protocol server on `127.0.0.1:8765` by default:

- Streamable HTTP: `http://127.0.0.1:8765/mcp`
- SSE: `http://127.0.0.1:8765/sse`
- Disable with `-mcp=false`; change with `-mcp-addr`; add `-mcp-token` for auth.

It exposes transport, song/track inspection, mixer, note editing, library,
harmony, export, settings and screenshots. See [docs/MCP.md](docs/MCP.md).

## Audio on Linux

gmid plays through miniaudio (malgo). Settings → Audio has an **Output
backend** (Auto / Pulse / ALSA / JACK), **Buffer size** (128–1024 frames) and
an **Output device** picker. Auto follows the system — normally PipeWire via
PulseAudio; ALSA talks to the PipeWire ALSA plugin or the hardware directly.
If playback crackles, raise the buffer to 512 or 1024, then try the other
backend. `GMID_AUDIO_LOG=1` prints backend and device diagnostics.

## License

GPL-3.0-or-later. Third-party assets and libraries are recorded in
`THIRD_PARTY_LICENSES.md`.
