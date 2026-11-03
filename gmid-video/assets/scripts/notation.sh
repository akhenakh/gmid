#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh" 8811
start_gmid || exit 1
drive gmid_settings_set '{"json":"{\"tempoWarnings\":false,\"sheet\":true,\"sheetHandColors\":true,\"sheetFollow\":true}"}'
drive gmid_open '{"path":"/home/akh/dev/gmid/assets/demo/Mozart - Rondo Alla Turca .mid"}'
hold 2
drive gmid_transport '{"action":"seek","time":8}'
drive gmid_transport '{"action":"play"}'
hold 8
# Plain black ink.
drive gmid_settings_set '{"json":"{\"sheetHandColors\":false}"}'
hold 4
# Hand colours back on.
drive gmid_settings_set '{"json":"{\"sheetHandColors\":true}"}'
hold 4
# Closing the sheet gives the roll its height back.
drive gmid_settings_set '{"json":"{\"sheet\":false}"}'
hold 3
drive gmid_settings_set '{"json":"{\"sheet\":true}"}'
hold 5

# Keep the process alive past the recorded clip so the studio sees a clean ending.
hold 90
