#!/usr/bin/env bash
# Silent DecentSampler performance: the narrator has finished, so the video
# carries the instrument's own output. The matching audio is exported offline
# and mixed into the final cut.
source "$(dirname "$0")/lib.sh" 8810
start_gmid || exit 1
drive gmid_open '{"path":"/home/akh/dev/gmid/assets/demo/Scott Joplin - Entertainer.mid"}'
drive gmid_decent '{"action":"load","path":"/home/akh/dev/gmid/assets/decent/demo-piano.dspreset"}'
hold 1
drive gmid_transport '{"action":"seek","time":0}'
drive gmid_transport '{"action":"play"}'
hold 90
