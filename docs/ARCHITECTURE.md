# gmid architecture

`gmid` is a single-process Go desktop app. `Orfeo/` is a read-only behavioural
reference and is not built or shipped.

## Packages

| Package | Responsibility |
| --- | --- |
| `cmd/gmid` | Flags, window, headless render entry point |
| `internal/config` | Preferences, paths, asset lookup |
| `internal/song` | Tick model, tempo/meter maps, beat grid, lead cut |
| `internal/smf` | SMF read/write, `GMID_*` metadata, history |
| `internal/theory` | GM names/groups, note names, chord vocabulary/detection, palettes |
| `internal/app` | Application state, transport, step-through, file loading |
| `internal/audio` | Timeline, realtime/offline engine, malgo output, DSP (later) |
| `internal/import` | MusicXML, Guitar Pro, Capella (later) |
| `internal/edit` | Note editor buffer and history (later) |
| `internal/library` | Folder scan, metadata cache, search (later) |
| `internal/midiio` | rtmidi input and diagnostics (later) |
| `ui` | Shell, roll, keyboard, mixer, explorers, settings, windows |

## Data flow

- The parser builds a `song.Song`: tick-based notes plus a tempo map, meter map,
  beat/bar grid and per-channel automation.
- The UI reads the model and the transport time; it never blocks on audio.
- Playback compiles the model into a time-sorted event timeline. The engine owns
  the audio clock and publishes time/level state back to the UI.
- Saves are rebuilt from the model as a new `_GMID_vN.mid`; originals are never
  modified.

## Rendering rules

Shirei rebuilds the container tree each frame. The roll culls notes to the
visible time window; bar lines are a small fixed set; the keyboard is 88 keys.
Hand-positioned content uses `Float` + `NoAnimate`. Complex art (circle of
fifths, starmap) is rasterized with `fogleman/gg` and displayed with `UseImage`.

## Invariants

See `.operator/specs/gmid.md` for the durable contract (file safety, metadata,
lead cut, mute/visibility semantics, loop state reset, tempo corrections).
