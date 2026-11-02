#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh" 8801
start_gmid || exit 1
drive gmid_settings_set '{"json":"{\"tempoWarnings\":false}"}'
drive gmid_open '{"path":"/home/akh/dev/gmid/assets/demo/Vivaldi 4 stagioni - Estate.mid"}'
hold 2
drive gmid_transport '{"action":"seek","time":14}'
drive gmid_transport '{"action":"play"}'
hold 7
drive gmid_transport '{"action":"speed","speed":0.7}'
hold 4
drive gmid_transport '{"action":"speed","speed":1.0}'
drive gmid_transport '{"action":"transpose","transpose":2}'
hold 4
drive gmid_transport '{"action":"transpose","transpose":0}'
drive gmid_loop '{"enabled":true,"start":26,"end":38}'
hold 5
drive gmid_loop '{"clear":true}'
drive gmid_ui '{"floating_keys":true}'
hold 6
drive gmid_ui '{"floating_keys":false}'
drive gmid_ui '{"side_panel":false}'
hold 6

# Keep the process alive past the recorded clip so the studio sees a clean ending.
hold 90
