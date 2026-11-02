#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh" 8803
start_gmid || exit 1
drive gmid_settings_set '{"json":"{\"tempoWarnings\":false}"}'
drive gmid_decent '{"action":"unload"}'
drive gmid_ui '{"library":true}'
hold 4
drive gmid_library '{"action":"scan"}'
hold 4
drive gmid_library '{"action":"star","path":"/home/akh/.config/gmid/library/Scott Joplin - Entertainer.mid"}'
hold 3
drive gmid_library '{"action":"load","path":"/home/akh/.config/gmid/library/Scott Joplin - Entertainer.mid"}'
hold 3
drive gmid_ui '{"library":false}'
drive gmid_transport '{"action":"play"}'
hold 5
drive gmid_ui '{"settings":true}'
hold 6
drive gmid_settings_set '{"json":"{\"chordPrompter\":true,\"tooltips\":true}"}'
hold 3
drive gmid_ui '{"settings":false}'
hold 4

# Keep the process alive past the recorded clip so the studio sees a clean ending.
hold 90
