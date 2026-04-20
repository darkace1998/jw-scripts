# Docker Usage

This project includes a Docker image that can run `jw-scripts` commands on a cron schedule.

## Build

```bash
docker build -t jw-scripts:latest .
```

## Pull from GitHub Container Registry

```bash
docker pull ghcr.io/darkace1998/jw-scripts:latest
```

Version-tagged images are also published for release tags (for example `v1.7.0`).

## Run

```bash
docker run --rm \
  -e CRON_SCHEDULE="0 */6 * * *" \
  -e JW_COMMAND="jwb-index --download --update --lang E /data" \
  -e RUN_ON_STARTUP=true \
  -v "$(pwd)/data:/data" \
  jw-scripts:latest
```

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `CRON_SCHEDULE` | `0 */6 * * *` | Cron expression for recurring runs |
| `JW_COMMAND` | `jwb-index --download --update --lang E /data` | Command executed on each cron trigger |
| `JW_WORKDIR` | `/data` | Working directory for command execution |
| `RUN_ON_STARTUP` | `true` | If `true`, executes one run before starting cron |
| `TZ` | `UTC` | Timezone used inside the container |

## Cron schedule explained

`CRON_SCHEDULE` uses the standard 5-field cron format:

```text
* * * * *
| | | | |
| | | | +-- day of week (0-6, Sun=0)
| | | +---- month (1-12)
| | +------ day of month (1-31)
| +-------- hour (0-23)
+---------- minute (0-59)
```

Supported syntax:

- `*` any value (for example every hour)
- `*/N` every N units (for example `*/15` every 15 minutes)
- `A,B` specific values (for example `1,15`)
- `A-B` ranges (for example `1-5`)

Quick examples:

- `0 */6 * * *` run every 6 hours
- `30 3 * * *` run daily at 03:30
- `0 6 * * 1-5` run at 06:00 on weekdays
- `0 0 1 * *` run on the first day of each month

Tips:

- Set `TZ` if you want local-time scheduling instead of UTC.
- Use `RUN_ON_STARTUP=true` to execute immediately when the container starts, then continue on the cron schedule.

## Common command examples

```bash
# Update JW Broadcasting videos every day at 03:30
JW_COMMAND="jwb-index --download --update --lang E /data"
CRON_SCHEDULE="30 3 * * *"
```

```bash
# Download music categories every 12 hours
JW_COMMAND="jwb-music --download --lang E /data/music"
CRON_SCHEDULE="0 */12 * * *"
```

```bash
# Download daily text PDFs every morning
JW_COMMAND="jwb-books --category daily-text --language E --format pdf --output /data/books"
CRON_SCHEDULE="0 6 * * *"
```

## Notes

- The container writes all downloaded/output files under `/data` by default.
- Always mount `/data` as a volume to persist files between container restarts.
- GitHub Actions workflow `.github/workflows/docker.yml` builds the image for PR validation and publishes to GHCR only on version tags (`v*`).
