#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh" 8807
start_gmid || exit 1
drive gmid_settings_set '{"json":"{\"tempoWarnings\":false}"}'
drive gmid_open '{"path":"/home/akh/dev/gmid/assets/demo/Scott Joplin - Entertainer.mid"}'
hold 2
drive gmid_transport '{"action":"seek","time":20}'
drive gmid_transport '{"action":"play"}'
hold 3
drive gmid_ui '{"export":true}'
hold 5
drive gmid_export_audio '{"variant":"heard","format":"wav"}'
hold 9
drive gmid_ui '{"export":false}'
hold 2
drive gmid_chord_chart '{}'
hold 8

# Keep the process alive past the recorded clip so the studio sees a clean ending.
hold 90
