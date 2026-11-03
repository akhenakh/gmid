#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh" 8806
start_gmid || exit 1
drive gmid_settings_set '{"json":"{\"tempoWarnings\":false,\"sheet\":false}"}'
drive gmid_open '{"path":"/home/akh/dev/gmid/assets/demo/Scott Joplin - Entertainer.mid"}'
hold 2
drive gmid_ui '{"settings":true}'
hold 2
drive gmid_decent '{"action":"load","path":"/home/akh/dev/gmid/assets/decent/demo-piano.dspreset"}'
hold 8
drive gmid_ui '{"settings":false}'
drive gmid_transport '{"action":"seek","time":0}'
drive gmid_transport '{"action":"play"}'
hold 9
drive gmid_transport '{"action":"speed","speed":0.6}'
hold 5
drive gmid_transport '{"action":"speed","speed":1}'
drive gmid_decent '{"action":"controls"}'
hold 5
drive gmid_transport '{"action":"seek","time":30}'
hold 4

# Keep the process alive past the recorded clip so the studio sees a clean ending.
hold 90
