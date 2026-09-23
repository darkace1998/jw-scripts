#!/bin/sh
set -eu

workdir="${JW_WORKDIR:-/data}"
self="$(readlink -f "$0")"

# run_once runs the configured command a single time. Scheduled runs call
# the entrypoint with "run-once".
run_once() {
  echo "[$(date -Iseconds)] Running command: ${JW_COMMAND}"
  cd "$workdir"
  status=0
  sh -c "${JW_COMMAND}" || status=$?
  if [ "$status" -eq 0 ]; then
    echo "[$(date -Iseconds)] Run completed"
  else
    echo "[$(date -Iseconds)] Run failed with exit code ${status}" >&2
  fi
  return "$status"
}

if [ "${1:-}" = "run-once" ]; then
  run_once
  exit $?
fi

# Drop root privileges so downloads are owned by PUID:PGID.
if [ "$(id -u)" = "0" ]; then
  puid="${PUID:-1000}"
  pgid="${PGID:-1000}"
  if [ "$puid" != "0" ]; then
    mkdir -p "$workdir"
    # Earlier images ran as root; hand existing downloads over once.
    if [ "$(stat -c %u "$workdir")" = "0" ]; then
      echo "Changing ownership of ${workdir} to ${puid}:${pgid} (one-time migration from root)"
      chown -R "${puid}:${pgid}" "$workdir" ||
        echo "warning: could not change ownership of ${workdir}; set PUID=0 PGID=0 to run as root" >&2
    fi
    export HOME=/tmp
    exec su-exec "${puid}:${pgid}" "$self" "$@"
  fi
fi

# Any other arguments are run as a one-off command, e.g.
# "docker run --rm IMAGE jwb-books --category bible --output /data/books"
if [ "$#" -gt 0 ]; then
  exec "$@"
fi

cron_schedule="${CRON_SCHEDULE:-0 */6 * * *}"
run_on_startup="${RUN_ON_STARTUP:-true}"
crontab="$(mktemp)"
printf '%s %s run-once\n' "$cron_schedule" "$self" > "$crontab"

case "$run_on_startup" in
  true|TRUE|1|yes|YES)
    # A failed first run (e.g. network not up yet) must not stop the
    # container before the schedule starts.
    run_once || echo "Initial run failed; continuing with the cron schedule" >&2
    ;;
esac

echo "Using CRON_SCHEDULE=${cron_schedule}"
exec /usr/local/bin/supercronic -passthrough-logs "$crontab"
