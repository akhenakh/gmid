#!/usr/bin/env bash
source "$(dirname "$0")/lib.sh" 8809
start_gmid || exit 1
drive gmid_settings_set '{"json":"{\"handAssignment\":true,\"handTags\":true,\"chordPrompter\":true,\"stepThrough\":true,\"tapTempo\":true,\"sheet\":false}"}'
drive gmid_open '{"path":"/home/akh/dev/gmid/assets/demo/Mozart - Rondo Alla Turca .mid"}'
hold 2
drive gmid_transport '{"action":"seek","time":6}'
hold 3
for _ in 1 2 3 4 5; do
  drive gmid_step '{"action":"next"}'
  hold 2
done
drive gmid_step '{"action":"next_bar"}'
hold 2
drive gmid_step '{"action":"prev"}'
hold 2
drive gmid_tap '{"action":"tap"}'
sleep 0.55
drive gmid_tap '{"action":"tap"}'
sleep 0.55
drive gmid_tap '{"action":"tap"}'
sleep 0.55
drive gmid_tap '{"action":"tap"}'
hold 2
drive gmid_tap '{"action":"clear"}'
hold 2
drive gmid_keyboard '{"action":"lock","key":60}'
drive gmid_keyboard '{"action":"lock","key":64}'
drive gmid_keyboard '{"action":"lock","key":67}'
hold 5
drive gmid_keyboard '{"action":"unlock"}'
hold 2
drive gmid_keyboard '{"action":"chord","keys":[60,64,67],"duration_ms":900}'
hold 4

# Keep the process alive past the recorded clip so the studio sees a clean ending.
hold 90
