#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh" 8808
start_gmid || exit 1
drive gmid_settings_set '{"json":"{\"tempoWarnings\":false}"}'
drive gmid_open '{"path":"/home/akh/dev/gmid/assets/demo/Vivaldi 4 stagioni - Estate.mid"}'
hold 2
drive gmid_settings_set '{"json":"{\"chordPrompter\":true}"}'
drive gmid_transport '{"action":"seek","time":18}'
drive gmid_transport '{"action":"play"}'
drive gmid_ui '{"presentation":true}'
hold 11
drive gmid_ui '{"presentation":false,"floating_keys":true}'
hold 8
drive gmid_ui '{"floating_keys":false,"playback_editor":true}'
hold 8
drive gmid_ui '{"playback_editor":false}'
hold 6

# Keep the process alive past the recorded clip so the studio sees a clean ending.
hold 90
