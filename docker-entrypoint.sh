#!/bin/sh
set -eu

cron_schedule="${CRON_SCHEDULE:-0 */6 * * *}"
run_on_startup="${RUN_ON_STARTUP:-true}"

cat > /usr/local/bin/run-download.sh <<'EOF'
#!/bin/sh
set -eu
echo "[$(date -Iseconds)] Running command: ${JW_COMMAND}"
cd "${JW_WORKDIR:-/data}"
sh -c "${JW_COMMAND}"
echo "[$(date -Iseconds)] Run completed"
EOF
chmod +x /usr/local/bin/run-download.sh

cat > /tmp/jw-scripts.cron <<EOF
${cron_schedule} /usr/local/bin/run-download.sh
EOF

case "$run_on_startup" in
  true|TRUE|1|yes|YES)
    /usr/local/bin/run-download.sh
    ;;
esac

echo "Using CRON_SCHEDULE=${cron_schedule}"
exec /usr/local/bin/supercronic -passthrough-logs /tmp/jw-scripts.cron
