#!/usr/bin/env bash
# Mix the DecentSampler performance into the silent "decent-play" scene of the
# final cut. videocreator records native app scenes video-only, so the app
# audio is exported separately with gmid and placed at the measured offset.
#
#   ./mix-app-audio.sh [music.wav]
#
# Re-run after a fresh vc_render. The script probes the performance scene: a
# silent cut is copied to render/gmid-video-silent.mkv, a mixed cut is re-mixed
# from that preserved copy, so the music is never doubled.
set -euo pipefail
cd "$(dirname "$0")"

MUSIC=${1:-assets/audio/entertainer-demo-piano.wav}
SILENT=render/gmid-video-silent.mkv
OUT=render/gmid-video.mkv

if [ ! -f "$MUSIC" ]; then
  echo "missing $MUSIC — export the song with the demo piano first" >&2
  exit 1
fi

read -r START DUR < <(awk '
  /id: decent-play/ {f=1}
  f && /start_seconds:/ {s=$2}
  f && /duration_seconds:/ {d=$2; if (s!="" && d!="") {print s, d; exit}}
' render/manifest.yaml)
[ -n "${START:-}" ] || { echo "decent-play not found in manifest" >&2; exit 1; }

# A fresh vc_render leaves the performance scene silent; a mixed cut is loud
# there. Detect which one we have so re-running never doubles the music and a
# re-render refreshes the preserved silent copy automatically.
PROBE=$(awk -v s="$START" 'BEGIN{printf "%.2f", s+2}')
LEVEL=$(ffmpeg -v info -ss "$PROBE" -t 3 -i "$OUT" -af volumedetect -f null - 2>&1 |
  sed -n 's/.*mean_volume: \(-\{0,1\}[0-9.]*\) dB.*/\1/p' | head -1)
if [ -z "${LEVEL:-}" ]; then
  echo "could not probe $OUT" >&2
  exit 1
fi
if awk -v l="$LEVEL" 'BEGIN{exit !(l < -60)}'; then
  cp "$OUT" "$SILENT" # fresh render
elif [ ! -f "$SILENT" ]; then
  echo "$OUT already has audio but $SILENT is missing; render again first" >&2
  exit 1
fi

# Video time where the recorded song is at 0. Measured from the transport clock
# in the clip (at clip 15s the clock read 14.31s); recalibrate after recording.
OFFSET=${OFFSET:-0.69}
VOLUME=${VOLUME:-0.8}
TRIM=$(awk -v o="$OFFSET" 'BEGIN{print (o<0 ? -o : 0)}')
LEAD=$(awk -v o="$OFFSET" 'BEGIN{print (o>0 ? o : 0)}')
MUSIC_SECS=$(awk -v d="$DUR" -v l="$LEAD" 'BEGIN{print d-l}')
DELAY_MS=$(awk -v s="$START" -v l="$LEAD" 'BEGIN{printf "%d", (s+l)*1000}')
FULL=$(ffprobe -v error -show_entries format=duration -of csv=p=0 "$SILENT")

ffmpeg -y -i "$SILENT" -i "$MUSIC" -filter_complex \
  "[1:a]aresample=48000,atrim=start=${TRIM}:duration=${MUSIC_SECS},asetpts=PTS-STARTPTS,volume=${VOLUME},adelay=${DELAY_MS}:all=1,apad=whole_dur=${FULL}[m];[0:a][m]amix=inputs=2:duration=first:normalize=0[a]" \
  -map 0:v -map "[a]" -c:v copy -c:a aac -b:a 192k "$OUT.tmp.mkv" -loglevel error
mv "$OUT.tmp.mkv" "$OUT"
echo "mixed $MUSIC into $OUT (scene $START +$DUR, offset ${OFFSET}s, volume ${VOLUME})"
