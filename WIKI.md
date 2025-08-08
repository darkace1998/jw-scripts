# JW Scripts Wiki

This page documents the command-line arguments for the `jwb-index` and `jwb-offline` commands.

## `jwb-index`

The `jwb-index` command is used to index and download media from jw.org.

| Flag | Shorthand | Default | Description |
|---|---|---|---|
| `--append` | | `false` | append to file instead of overwriting |
| `--category` | `-c` | `VideoOnDemand` | comma separated list of categories to index |
| `--checksum` | | `false` | validate MD5 checksums |
| `--clean-symlinks` | | `false` | remove all old symlinks (mode=filesystem) |
| `--download` | `-d` | `false` | download media files |
| `--download-subtitles` | | `false` | download VTT subtitle files |
| `--exclude` | | `VODSJJMeetings` | comma separated list of categories to skip |
| `--fix-broken` | | `false` | check existing files and re-download them if they are broken |
| `--free` | | `0` | disk space in MiB to keep free |
| `--friendly` | `-H` | `false` | save downloads with human readable names |
| `--hard-subtitles` | | `false` | prefer videos with hard-coded subtitles |
| `--import` | | `""` | import of media files from this directory (offline) |
| `--lang` | `-l` | `E` | language code |
| `--languages` | `-L` | `false` | display a list of valid language codes |
| `--latest` | | `false` | index the "Latest Videos" category only |
| `--limit-rate` | `-R` | `1.0` | maximum download rate, in megabytes/s |
| `--list-categories` | `-C` | `""` | print a list of (sub) category names |
| `--mode` | `-m` | `""` | output mode (filesystem, html, m3u, run, stdout, txt) |
| `--no-warning` | | `true` | do not warn when space limit seems wrong |
| `--quality` | `-Q` | `720` | maximum video quality |
| `--quiet` | `-q` | `0` | less info, can be used multiple times |
| `--since` | | `0` | only index media newer than this date (YYYY-MM-DD) |
| `--sort` | | `""` | sort output (newest, oldest, name, random) |
| `--update` | | `false` | update existing categories with the latest videos |

## `jwb-offline`

The `jwb-offline` command is used to shuffle and play videos in a directory.

| Flag | Default | Description |
|---|---|---|
| `--replay-sec` | `30` | seconds to replay after a restart |
| `--cmd` | `omxplayer --pos {} --no-osd` | video player command |
| `--quiet` | `0` | less info, can be used multiple times |
