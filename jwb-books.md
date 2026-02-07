# JWB Books Wiki

This page documents the command-line arguments and usage for the `jwb-books` command.

## Overview

`jwb-books` is a command-line tool for downloading JW.org publications in multiple languages and formats. It supports downloading Bibles, Daily Texts, Yearbooks, Convention Materials, Magazines, and more.

## Usage

```bash
# Show help
jwb-books --help

# List all supported languages
jwb-books --list-languages

# List all supported formats
jwb-books --list-formats

# List available categories
jwb-books --list-categories

# Download the Daily Text in English (PDF)
jwb-books --category daily-text --language E --format pdf

# Download the Bible in Spanish (EPUB)
jwb-books --category bible --language S --format epub

# Search for publications
jwb-books --search "daily" --language F
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--category` | `""` | Category to download (use `--list-categories` to see options) |
| `--format` | `pdf` | Format to download (use `--list-formats` to see options) |
| `--help` | `false` | Show help information |
| `--language` | `E` | Language code (use `--list-languages` to see options) |
| `--list-categories` | `false` | List all available categories |
| `--list-formats` | `false` | List all supported formats |
| `--list-languages` | `false` | List all supported languages |
| `--output` | `downloads` | Output directory for downloads |
| `--search` | `""` | Search for publications |

## Categories

The following publication categories are available:

| Category Key | Name | Description | Publications |
|---|---|---|---|
| `bible` | Bible | New World Translation of the Holy Scriptures | nwtsty |
| `daily-text` | Daily Text | Examining the Scriptures Daily | es25 |
| `yearbooks` | Yearbooks | Watch Tower Publications Index and Yearbooks | dx24 |
| `circuit-assembly` | Circuit Assembly Programs | Circuit Assembly Programs | ca-brpgm26 |
| `convention` | Convention Materials | Convention invitations and programs | co-inv25 |
| `magazines` | Magazines | Watchtower and Awake! magazines | w, g |

To see all available categories for a specific language:

```bash
jwb-books --list-categories --language S
```

## Formats

The following download formats are supported:

| Format | Description |
|---|---|
| `pdf` | Portable Document Format |
| `epub` | Electronic Publication (e-readers) |
| `mp3` | Audio (MP3) |
| `mp4` | Video (MP4) |
| `rtf` | Rich Text Format |
| `brl` | Braille |

Not all publications are available in all formats. Use `--list-formats` to see the complete list.

## Languages

Common language codes:

| Code | Language |
|---|---|
| `E` | English |
| `S` | español (Spanish) |
| `F` | Français (French) |
| `T` | Português (Brazilian Portuguese) |
| `X` | Deutsch (German) |
| `P` | polski (Polish) |
| `Z` | Svenska (Swedish) |
| `J` | 日本語 (Japanese) |
| `K` | українська (Ukrainian) |
| `I` | Italiano (Italian) |
| `U` | русский (Russian) |
| `A` | العربية (Arabic) - RTL |
| `Q` | עברית (Hebrew) - RTL |
| `B` | čeština (Czech) |
| `C` | hrvatski (Croatian) |
| `D` | Dansk (Danish) |
| `G` | Ελληνική (Greek) |
| `H` | magyar (Hungarian) |
| `L` | lietuvių (Lithuanian) |
| `M` | Română (Romanian) |
| `N` | Norsk (Norwegian) |
| `O` | Nederlands (Dutch) |
| `V` | slovenčina (Slovak) |
| `W` | Cymraeg (Welsh) |

For a complete list of supported languages, run:

```bash
jwb-books --list-languages
```

## Examples

### Download the Bible in PDF format (English)
```bash
jwb-books --category bible --language E --format pdf
```

### Download the Bible in EPUB format (Spanish)
```bash
jwb-books --category bible --language S --format epub
```

### Download Daily Text in German
```bash
jwb-books --category daily-text --language X
```

### Download to a specific directory
```bash
jwb-books --category daily-text --output /path/to/downloads
```

### Search for publications containing "daily"
```bash
jwb-books --search "daily" --language E
```

### Download convention materials in French
```bash
jwb-books --category convention --language F --format pdf
```

### Download yearbooks in Portuguese
```bash
jwb-books --category yearbooks --language T
```

### List categories available in Japanese
```bash
jwb-books --list-categories --language J
```
