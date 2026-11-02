#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh" 8805
start_gmid || exit 1
drive gmid_settings_set '{"json":"{\"tempoWarnings\":false}"}'
drive gmid_open '{"path":"/home/akh/dev/gmid/assets/demo/Vivaldi 4 stagioni - Estate.mid"}'
hold 2
drive gmid_transport '{"action":"seek","time":0.5}'
drive gmid_ui '{"note_editor":true}'
hold 2
drive gmid_note_edit '{"action":"enter","track":0,"tool":"select"}'
drive gmid_note_edit '{"action":"add","key":67,"tick":1920,"duration_ticks":960,"velocity":98}'
hold 3
drive gmid_note_edit '{"action":"move","delta_ticks":240,"delta_key":2}'
hold 3
drive gmid_note_edit '{"action":"add","key":72,"tick":2880,"duration_ticks":480,"velocity":80}'
drive gmid_note_edit '{"action":"velocity","velocity":120}'
hold 3
drive gmid_note_edit '{"action":"undo"}'
hold 2
drive gmid_note_edit '{"action":"redo"}'
hold 3
drive gmid_note_edit '{"action":"exit"}'
drive gmid_ui '{"note_editor":false,"playback_editor":true}'
hold 8
drive gmid_ui '{"playback_editor":false,"note_editor":true}'
hold 5
drive gmid_ui '{"note_editor":false}'
hold 3

# Keep the process alive past the recorded clip so the studio sees a clean ending.
hold 90
