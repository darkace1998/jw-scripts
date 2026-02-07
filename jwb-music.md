# JWB Music Wiki

This page documents the command-line arguments and usage for the `jwb-music` command.

## Overview

`jwb-music` is a specialized tool for downloading all music and audio files from jw.org. It downloads from all music-related categories including Original Songs, "Sing Out Joyfully" (Meetings, Vocals, Instrumental), Children's Songs, and Kingdom Melodies.

## Usage

```bash
# Download all music files (default behavior)
jwb-music

# Download to a specific directory
jwb-music /path/to/music

# Download JW Broadcasting monthly programs as MP3
jwb-music -c JWBroadcasting

# Download only Kingdom Melodies in Spanish
jwb-music -c KingdomMelodies -l S

# List available music categories
jwb-music --list-categories

# List available language codes
jwb-music -L
```

## Flags

| Flag | Shorthand | Default | Description |
|---|---|---|---|
| `--append` | | `false` | append to file instead of overwriting |
| `--audio-only` | | `true` | download only audio (MP3) files, skip video-only content (enabled by default) |
| `--category` | `-c` | all music categories | comma separated list of music categories to include |
| `--checksum` | | `false` | validate MD5 checksums |
| `--download` | `-d` | `true` | download music files (enabled by default) |
| `--exclude` | | `""` | comma separated list of categories to skip |
| `--fix-broken` | | `false` | check existing files and re-download them if they are broken |
| `--free` | | `0` | disk space in MiB to keep free |
| `--friendly` | `-H` | `false` | save downloads with human readable names |
| `--import` | | `""` | import of music files from this directory (offline) |
| `--lang` | `-l` | `E` | language code |
| `--languages` | `-L` | `false` | display a list of valid language codes |
| `--limit-rate` | `-R` | `25.0` | maximum download rate, in megabytes/s |
| `--list-categories` | | `false` | list all available music categories |
| `--mode` | `-m` | `""` | output mode (filesystem, html, m3u, run, stdout, txt) |
| `--no-warning` | | `true` | do not warn when space limit seems wrong |
| `--quiet` | `-q` | `0` | less info, can be used multiple times |
| `--safe-filenames` | | `false` (Windows: `true`) | use filesystem-safe filenames (automatically enabled on Windows) |
| `--since` | | `0` | only index music newer than this date (YYYY-MM-DD) |
| `--sort` | | `""` | sort output (newest, oldest, name, random) |
| `--update` | | `false` | update existing categories with the latest music |

## Music Categories

The following music categories are available for download:

| Category Code | Description |
|---|---|
| `AudioOriginalSongs` | Original Songs |
| `SJJMeetings` | Sing Out Joyfully - Meeting Songs |
| `SJJChorus` | Sing Out Joyfully - Vocals/Chorus |
| `SJJInstrumental` | Sing Out Joyfully - Instrumental |
| `AudioChildrenSongs` | Children's Songs |
| `KingdomMelodies` | Kingdom Melodies |
| `JWBroadcasting` | JW Broadcasting Monthly Programs (MP3 audio) |

By default, all music categories (except `JWBroadcasting`) are downloaded. To download JW Broadcasting audio, explicitly include it:

```bash
jwb-music -c JWBroadcasting
```

Or download everything including JW Broadcasting:

```bash
jwb-music -c AudioOriginalSongs,SJJMeetings,SJJChorus,SJJInstrumental,AudioChildrenSongs,KingdomMelodies,JWBroadcasting
```

## Output Modes

| Mode | Description |
|---|---|
| `filesystem` | Save media files to disk |
| `html` | Generate HTML playlist |
| `m3u` | Generate M3U playlist file |
| `run` | Play media directly |
| `stdout` | Output URLs to stdout |
| `txt` | Generate text file list |

## Examples

### Download all music in English (default)
```bash
jwb-music
```

### Download Kingdom Melodies only
```bash
jwb-music -c KingdomMelodies
```

### Download children's songs in Spanish
```bash
jwb-music -c AudioChildrenSongs -l S
```

### Create an M3U playlist without downloading
```bash
jwb-music -d=false -m m3u
```

### Download with human-readable filenames
```bash
jwb-music -H
```

### Download and keep 1GB free disk space
```bash
jwb-music --free 1024
```

### Update existing collection with latest music
```bash
jwb-music --update
```

### Download JW Broadcasting audio programs
```bash
jwb-music -c JWBroadcasting
```

## Languages

For a full list of available language codes, run:

```bash
jwb-music -L
```

Or see the [Languages section in WIKI.md](WIKI.md#languages) for the complete list of language codes.
