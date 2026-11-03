# Third-party licenses and assets

`gmid` is GPL-3.0-or-later. The following components retain their own licenses.

## Go modules

| Module | License |
| --- | --- |
| `go.hasen.dev/shirei` | zlib-style permissive (see module `COPYING`) |
| `gitlab.com/gomidi/midi/v2` | MIT |
| thestk/rtmidi (bundled by `rtmididrv`) | MIT |
| `github.com/sinshu/go-meltysynth` | MIT |
| `github.com/gen2brain/malgo` (miniaudio) | public domain / MIT |
| `github.com/fogleman/gg` | MIT |
| `github.com/chinmay-sawant/gowkhtmltopdf` | MIT |
| `github.com/jfreymuth/oggvorbis` | MIT |
| `github.com/fsnotify/fsnotify` | BSD-3-Clause |
| `github.com/braheezy/shine-mp3` (encoder fork) | MIT |

## Fonts

| Asset | License |
| --- | --- |
| Inter (Regular, Bold) | SIL Open Font License 1.1 |
| JetBrains Mono (Regular) | SIL Open Font License 1.1 |
| Typicons (bundled by Shirei) | SIL Open Font License 1.1 |

## Soundfont and demo content

- `GeneralUser-GS.sf2` is not committed to this repository. `make assets`
  downloads it from the upstream [GeneralUser-GS repository](https://github.com/mrbumpy409/GeneralUser-GS)
  (pinned to a commit) into `assets/soundfonts/`. GeneralUser GS is distributed
  free of charge by S. Christian Collins; see its accompanying terms.
- Demo MIDI files are arrangements of public-domain compositions used for local
  testing. They are not required to build or run the app.

## Reference implementation

The feature set and several algorithms are inspired by
[Orfeo](https://github.com/Squarebow/Orfeo) (GPL-3.0-or-later). `gmid` is an
independent Go implementation, not a port of its Electron/React code.
