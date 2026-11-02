#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh" 8802
start_gmid || exit 1
drive gmid_settings_set '{"json":"{\"tempoWarnings\":false}"}'
drive gmid_open '{"path":"/home/akh/dev/gmid/assets/demo/Vivaldi 4 stagioni - Estate.mid"}'
hold 2
drive gmid_transport '{"action":"seek","time":20}'
drive gmid_transport '{"action":"play"}'
hold 2
drive gmid_ui '{"mixer":true}'
hold 2
drive gmid_mixer '{"action":"set","channel":0,"volume":0.35,"pan":0.2,"reverb":0.7}'
hold 3
drive gmid_mixer '{"action":"set","channel":1,"volume":1.2,"pan":0.8,"chorus":0.4}'
drive gmid_mixer '{"action":"set","master_volume":0.9,"master_tone":0.62,"compressor":true}'
hold 4
drive gmid_ui '{"mixer":false}'
hold 3
drive gmid_set_track '{"index":2,"muted":true}'
hold 4
drive gmid_set_track '{"index":2,"muted":false}'
drive gmid_set_track '{"index":4,"visible":false}'
hold 4
drive gmid_focus '{"action":"on"}'
hold 5
drive gmid_focus '{"action":"off"}'
hold 6

# Keep the process alive past the recorded clip so the studio sees a clean ending.
hold 90
