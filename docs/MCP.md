# gmid MCP server

gmid exposes a Model Context Protocol server so an LLM client can inspect and
drive the running app: transport, tracks, mixer, notes, library, harmony,
editors, export and screenshots.

## Start

```sh
./bin/gmid                 # MCP on by default at 127.0.0.1:8765
./bin/gmid -mcp=false      # disable
./bin/gmid -mcp-addr 127.0.0.1:9000
./bin/gmid -mcp-token secret
```

Endpoints:

| Transport | URL |
| --- | --- |
| Streamable HTTP | `http://127.0.0.1:8765/mcp` |
| SSE | `http://127.0.0.1:8765/sse` (messages at `/message`) |
| Info | `http://127.0.0.1:8765/info` |

The server binds to loopback by default. If `-mcp-token` is set, send
`Authorization: Bearer <token>` (or `?token=<token>` for SSE).

## Smoke test

```sh
go run ./tools/mcpcheck -list
go run ./tools/mcpcheck -tool gmid_status
go run ./tools/mcpcheck -tool gmid_open -args '{"path":"/music/song.mid"}'
go run ./tools/mcpcheck -tool gmid_screenshot -out /tmp/gmid.png
```

## Tools

| Tool | Purpose |
| --- | --- |
| `gmid_status` | App/transport/audio/MIDI summary; `verbose` includes full tracks. |
| `gmid_song` | Key, tempo, bars and every track of the open song. |
| `gmid_open` | Load a MIDI/MusicXML/KAR/Guitar Pro/Capella file. |
| `gmid_transport` | `play`, `pause`, `toggle`, `stop`, `seek`, `speed`, `transpose`, `status`. |
| `gmid_loop` | Set, clear or inspect the loop region. |
| `gmid_metronome` | Click on/off, volume, count-in and bars. |
| `gmid_mixer` | Read/change channel faders, pan, chorus, reverb and master strip. |
| `gmid_set_track` | Rename, recolour, mute/solo/eye, keyboard lighting, program, exclude. |
| `gmid_track_edit` | Merge a track into the previous one, or split hands. |
| `gmid_notes` | Paginated note query by track, time and pitch range. |
| `gmid_note_edit` | Enter/exit the editor; select, add, move, resize, velocity, delete, undo/redo, save. |
| `gmid_library` | Scan, search, star/unstar and load songs. |
| `gmid_chords` | Chord at the playhead, previous/current/next, or per bar. |
| `gmid_scale` | List scale types or diatonic chords for a scale. |
| `gmid_chord` | Build any chord, including inversions. |
| `gmid_progressions` | List the nine progressions or resolve one in a key. |
| `gmid_focus` | Focus mode on/off/status. |
| `gmid_tap` | Tap the beat, inspect or clear the tapped grid. |
| `gmid_keyboard` | Press/release keys, audition chords, clear or lock a chord. |
| `gmid_ui` | Open/close panels: settings, library, mixer, editors, explorers, presentation, floating keys. |
| `gmid_settings` / `gmid_settings_set` | Read preferences or merge a partial JSON update. |
| `gmid_step` | Walk the song note/chord or bar at a time (`next`, `prev`, `next_bar`, `prev_bar`). |
| `gmid_decent` | Status, load, unload or inspect a DecentSampler `.dspreset`/`.dslibrary`; `gain_db` sets the instrument gain. |
| `gmid_save_version` | Write a new `_GMID_vN.mid` with an embedded history entry. |
| `gmid_export_audio` / `gmid_export_status` | Render WAV/MP3 (original, focus or heard). |
| `gmid_chord_chart` | Generate the chord-chart PDF. |
| `gmid_midi_devices` | List hardware input ports and event count. |
| `gmid_screenshot` | Render the current UI and return it as an MCP image. |

## Notes

- State changes apply to the live app; the UI updates on the next frame.
- Times are song seconds. Ticks are MIDI ticks at the song's PPQ.
- MIDI note IDs come from `gmid_notes`; the editor's `gmid_note_edit` accepts them.
- `gmid_settings_set` merges JSON; unknown keys are ignored by `encoding/json`.
- The server holds Shirei's frame lock while reading/writing app state, so MCP
  calls cannot race the render loop. Screenshots render through the same lock.
