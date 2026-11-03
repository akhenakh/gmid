#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh" 8804
start_gmid || exit 1
drive gmid_settings_set '{"json":"{\"tempoWarnings\":false,\"sheet\":false}"}'
drive gmid_open '{"path":"/home/akh/dev/gmid/assets/demo/Scott Joplin - Entertainer.mid"}'
hold 2
drive gmid_transport '{"action":"seek","time":16}'
drive gmid_transport '{"action":"play"}'
hold 4
drive gmid_ui '{"chord_explorer":true}'
hold 11
drive gmid_ui '{"chord_explorer":false,"scale_explorer":true}'
hold 11
drive gmid_ui '{"scale_explorer":false}'
drive gmid_keyboard '{"action":"chord","keys":[60,64,67,71],"duration_ms":900}'
hold 3

# Keep the process alive past the recorded clip so the studio sees a clean ending.
hold 90
