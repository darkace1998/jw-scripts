# JW Scripts Command Reference

This page documents the command-line arguments for the `jwb-index` and `jwb-offline` commands.

All commands exit with a non-zero status when anything failed (a category that could not be indexed, a failed download, ...), so scripts, cron jobs and containers can detect problems. Work that can still be done is completed first.

## `jwb-index`

The `jwb-index` command is used to index and download media from jw.org.

| Flag | Shorthand | Default | Description |
|---|---|---|---|
| `--append` | | `false` | append to file instead of overwriting |
| `--audio-only` | | `false` | download only audio (MP3) files, skip video-only content |
| `--category` | `-c` | `VideoOnDemand` | comma separated list of categories to index |
| `--checksum` | | `false` | verify MD5 checksums of downloads (and of existing files with `--fix-broken`) |
| `--clean-symlinks` | | `false` | remove all old symlinks (mode=filesystem) |
| `--download` | `-d` | `false` | download media files |
| `--download-subtitles` | | `false` | download VTT subtitle files |
| `--exclude` | | `VODSJJMeetings` | comma separated list of categories to skip |
| `--fix-broken` | | `false` | check the size (and MD5 with `--checksum`) of existing files and re-download broken ones |
| `--free` | | `0` | disk space in MiB to keep free |
| `--friendly` | `-H` | `false` | save downloads with human readable names |
| `--hard-subtitles` | | `false` | prefer videos with hard-coded subtitles |
| `--import` | | `""` | copy media files from this directory into the library (offline import) |
| `--lang` | `-l` | `E` | language code |
| `--languages` | `-L` | `false` | display a list of valid language codes |
| `--latest` | `-D` | `false` | fetch subtitles and videos from the past 31 days up to today (31-day window ending today) |
| `--limit-rate` | `-R` | `25.0` | maximum download rate, in megabytes/s |
| `--list-categories` | `-C` | `""` | print a list of (sub) category names |
| `--list-categories-all` | | `false` | list all available root categories |
| `--metadata` | | `false` | write an `.nfo` metadata file next to each download (read by Jellyfin, Emby and Kodi; Plex via an NFO agent), see [Metadata files](#metadata-files) |
| `--mode` | `-m` | `""` | output mode (filesystem, html, m3u, run, stdout, txt; `txtmulti`, `m3umulti` and `htmlmulti` write one file per category) |
| `--output` | `-o` | `""` | output filename for txt/m3u/html modes |
| `--no-warning` | | `false` | do not warn when the disk space limit (`--free`) seems wrong |
| `--quality` | `-Q` | `720` | maximum video quality |
| `--quiet` | `-q` | `0` | less info, can be repeated (`-q`, `-qq`) or given a level (`--quiet=2`) |
| `--safe-filenames` | | `false` (Windows: `true`) | use filesystem-safe filenames (automatically enabled on Windows) |
| `--since` | | `""` | only index media newer than this date (YYYY-MM-DD) |
| `--sort` | | `""` | sort output (newest, oldest, name, random) |
| `--update` | | `false` | update existing categories with the latest videos |

## `jwb-offline`

The `jwb-offline` command is used to shuffle and play videos in a directory.

| Flag | Default | Description |
|---|---|---|
| `--replay-sec` | `30` | seconds to replay after a restart |
| `--cmd` | `mpv --start {} --no-terminal` | video player command (`{}` is replaced with the start position in seconds) |
| `--quiet` | `0` | less info, can be repeated (`-q`, `-qq`) or given a level (`--quiet=2`) |

On Ctrl+C or `SIGTERM` the player is stopped and the position is saved in `dump.json`, so playback resumes there next time.

## Downloads and filenames

- Each media item is downloaded once, even when it appears in several categories.
- With `--friendly`, media with the same title get a ` (1)`, ` (2)`, ... suffix. The oldest item keeps the plain name, so names stay the same between runs.
- Subtitles are named after their video (`<video name>.vtt`) so players and media servers pick them up automatically.
- Downloads go to a `.part` file and are only moved into place once complete and verified (size, and MD5 with `--checksum`). Interrupted downloads resume on the next run; a download that receives no data for 2 minutes is aborted.
- Downloaded files and directories are readable by other users, so a media server running as a different user can read them.
- `--import DIR` copies the media files from `DIR` into the download directory.

## Metadata files

With `--metadata`, every downloaded file gets an NFO file with the same name (`Some Video.mp4` gets `Some Video.nfo`). The media files themselves are never modified.

NFO is the XML metadata format from Kodi:

- **Jellyfin** and **Emby** read it natively; check that the *Nfo* metadata reader is enabled in the library settings.
- **Kodi** reads it natively.
- **Plex** needs an NFO agent such as [XBMCnfoMoviesImporter](https://github.com/gboudreau/XBMCnfoMoviesImporter.bundle).

Each NFO describes the file as a `<movie>` with its title, release date (`<premiered>`, `<releasedate>`, `<year>`), runtime, category (`<genre>`), `jw.org` as studio and the source URL as its ID. Use a *Movies* or *Home videos* library.

Media servers read metadata for **music** only from tags inside the audio files, not from NFO files, so for MP3 downloads the NFO files are informational. NFO files for publications (PDF, EPUB, ...) also contain the publication code, issue and format.

Unchanged NFO files are not rewritten. JSON metadata files (`<filename>.json`) written by earlier versions are removed. Tags that earlier versions embedded in MP3/MP4 files are left as they are.

## Languages

The following is a list of available languages and their codes that can be used with the `--lang` flag.

| Code | Name |
|---|---|
| ABK | Abkhaz |
| AF | Afrikaans |
| AL | Albanian |
| ALT | Altai |
| ALU | Alur |
| ASL | American Sign Language |
| AM | Amharic |
| LAS | Angolan Sign Language |
| A | Arabic |
| AEY | Arabic (Egypt) |
| LSA | Argentinean Sign Language |
| REA | Armenian |
| AKN | Aukan |
| AUS | Australian Sign Language |
| OGS | Austrian Sign Language |
| AP | Aymara |
| AJR | Azerbaijani |
| BAK | Bashkir |
| BQ | Basque |
| BS | Bassa (Cameroon) |
| AK | Batak (Karo) |
| BT | Batak (Toba) |
| BZK | Belize Kriol |
| BE | Bengali |
| IK | Biak |
| BI | Bicol |
| LM | Bislama |
| BVL | Bolivian Sign Language |
| WSL | Botswana Sign Language |
| BO | Boulou |
| LSB | Brazilian Sign Language |
| BSL | British Sign Language |
| BL | Bulgarian |
| CB | Cambodian |
| AN | Catalan |
| CV | Cebuano |
| CN | Chichewa |
| SCH | Chilean Sign Language |
| CNS | Chinese Cantonese (Simplified) |
| CHC | Chinese Cantonese (Traditional) |
| CHS | Chinese Mandarin (Simplified) |
| CH | Chinese Mandarin (Traditional) |
| CSL | Chinese Sign Language |
| CG | Chitonga |
| CT | Chitonga (Malawi) |
| TB | Chitumbuka |
| CK | Chokwe |
| CHL | Chol |
| CU | Chuvash |
| CW | Cibemba |
| NM | Cinamwanga |
| CIN | Cinyanja |
| LSC | Colombian Sign Language |
| C | Croatian |
| CBS | Cuban Sign Language |
| B | Czech |
| CSE | Czech Sign Language |
| DMR | Damara |
| DG | Dangme |
| D | Danish |
| DSL | Danish Sign Language |
| DAR | Dari |
| DA | Douala |
| LF | Drehu |
| O | Dutch |
| SEC | Ecuadorian Sign Language |
| ED | Edo |
| EF | Efik |
| EMB | Emberá (Catío) |
| E | English |
| ST | Estonian |
| STD | Estonian Sign Language |
| ESL | Ethiopian Sign Language |
| EW | Ewe |
| EWN | Ewondo |
| FGN | Fang |
| FN | Fijian |
| FSL | Filipino Sign Language |
| FI | Finnish |
| FID | Finnish Sign Language |
| F | French |
| LSF | French Sign Language |
| GA | Ga |
| GLC | Galician |
| GRF | Garifuna |
| GE | Georgian |
| X | German |
| DGS | German Sign Language |
| GHM | Ghomálá’ |
| G | Greek |
| GSL | Greek Sign Language |
| GI | Guarani |
| GU | Gujarati |
| EG | Gun |
| CR | Haitian Creole |
| HMA | Hamshen (Armenian) |
| HMS | Hamshen (Cyrillic) |
| HA | Hausa |
| HW | Hawaiian |
| Q | Hebrew |
| HR | Herero |
| HV | Hiligaynon |
| HI | Hindi |
| MO | Hiri Motu |
| HM | Hmong (White) |
| H | Hungarian |
| HDF | Hungarian Sign Language |
| HSK | Hunsrik |
| IA | Iban |
| IG | Ibanag |
| IBI | Ibinda |
| IC | Icelandic |
| IB | Igbo |
| IL | Iloko |
| INS | Indian Sign Language |
| IN | Indonesian |
| INI | Indonesian Sign Language |
| IS | Isoko |
| I | Italian |
| ISL | Italian Sign Language |
| J | Japanese |
| JSL | Japanese Sign Language |
| JA | Javanese |
| KBR | Kabardin-Cherkess |
| KBV | Kabuverdianu |
| KBY | Kabyle |
| KB | Kamba |
| KA | Kannad |
| KR | Karen (S'gaw) |
| AZ | Kazakh |
| GK | Kekchi |
| KD | Kikaonde |
| KG | Kikongo |
| KQ | Kikuyu |
| KU | Kiluba |
| KIM | Kimbundu |
| YW | Kinyarwanda |
| KZ | Kirghiz |
| GB | Kiribati |
| RU | Kirundi |
| KI | Kisi |
| KSN | Kisonge |
| MK | Kongo |
| KO | Korean |
| KSL | Korean Sign Language |
| KRI | Krio |
| RDU | Kurdish Kurmanji (Caucasus) |
| RDC | Kurdish Kurmanji (Cyrillic) |
| WG | Kwangali |
| KY | Kwanyama |
| LT | Latvian |
| LI | Lingala |
| L | Lithuanian |
| LWX | Low German |
| LU | Luganda |
| LD | Lunda |
| LO | Luo |
| LV | Luvale |
| LX | Luxembourgish |
| MC | Macedonian |
| TTM | Madagascar Sign Language |
| MG | Malagasy |
| MSL | Malawi Sign Language |
| ML | Malay |
| MY | Malayalam |
| MT | Maltes |
| MZ | Mam |
| MWL | Mambwe-Lungu |
| MPD | Mapudungun |
| CE | Mauritian Creole |
| MAY | Maya |
| MAZ | Mazatec (Huautla) |
| DU | Medumba |
| LSM | Mexican Sign Language |
| MGL | Mingrelian |
| MXG | Mixtec (Guerrero) |
| KHA | Mongolian |
| MTU | Motu |
| BU | Myanmar |
| NHC | Nahuatl (Central) |
| NHG | Nahuatl (Guerrero) |
| NHH | Nahuatl (Huasteca) |
| NHT | Nahuatl (Northern Puebla) |
| NV | Navajo |
| NBL | Ndebele |
| NBZ | Ndebele (Zimbabwe) |
| OD | Ndonga |
| NP | Nepali |
| NGB | Ngabere |
| NGL | Ngangela |
| NMB | Ngiemboon |
| NI | Nias |
| NGP | Nigerian Pidgin |
| N | Norwegian |
| NDF | Norwegian Sign Language |
| NEN | Nsenga (Zambia) |
| NK | Nyaneka |
| OSS | Ossetian |
| OT | Otetela |
| OTM | Otomi (Mezquital Valley) |
| PN | Pangasinan |
| PAA | Papiamento (Aruba) |
| PA | Papiamento (Curaçao) |
| LSP | Paraguayan Sign Language |
| PH | Pashto |
| PR | Persian |
| SPE | Peruvian Sign Language |
| PCM | Pidgin (Cameroon) |
| PGW | Pidgin (West Africa) |
| P | Polish |
| PDF | Polish Sign Language |
| PMR | Pomeranian |
| T | Portuguese (Brazil) |
| TPO | Portuguese (Portugal) |
| LGP | Portuguese Sign Language |
| PJ | Punjabi |
| PJN | Punjabi (Shahmukhi) |
| LSQ | Quebec Sign Language |
| QUN | Quechua (Ancash) |
| QUA | Quechua (Ayacucho) |
| QUB | Quechua (Bolivia) |
| QU | Quechua (Cuzco) |
| QUL | Quechua (Huallaga Huánuco) |
| QIC | Quichua (Chimborazo) |
| QII | Quichua (Imbabura) |
| M | Romanian |
| LMG | Romanian Sign Language |
| RM | Romany (Macedonia) |
| RMC | Romany (Macedonia) Cyrillic |
| RMG | Romany (Southern Greece) |
| RMV | Romany (Vlax, Russia) |
| U | Russian |
| RSL | Russian Sign Language |
| SM | Samoan |
| SG | Sango |
| SRM | Saramaccan |
| SE | Sepedi |
| SPL | Sepulana |
| SB | Serbian (Cyrillic) |
| SBO | Serbian (Roman) |
| SU | Sesotho (Lesotho) |
| SSA | Sesotho (South Africa) |
| TN | Setswana |
| SC | Seychelles Creole |
| CA | Shona |
| SK | Silozi |
| SN | Sinhala |
| V | Slovak |
| VSL | Slovak Sign Language |
| SV | Slovenian |
| SP | Solomon Islands Pidgin |
| SAS | South African Sign Language |
| S | Spanish |
| LSE | Spanish Sign Language |
| SR | Sranantongo |
| SD | Sunda |
| SW | Swahili |
| ZS | Swahili (Congo) |
| SWI | Swati |
| Z | Swedish |
| SSL | Swedish Sign Language |
| XSW | Swiss German |
| SGS | Swiss German Sign Language |
| TG | Tagalog |
| TH | Tahitian |
| TSL | Taiwanese Sign Language |
| TJ | Tajik |
| TAL | Talian |
| TL | Tamil |
| TND | Tandroy |
| TNK | Tankarana |
| TRS | Tarascan |
| TAT | Tatar |
| TU | Telugu |
| TTP | Tetun Dili |
| SI | Thai |
| SIL | Thai Sign Language |
| TV | Tiv |
| TLN | Tlapanec |
| TJO | Tojolabal |
| MP | Tok Pisin |
| TO | Tongan |
| TOT | Totonac |
| SH | Tshiluba |
| TS | Tsonga |
| TK | Turkish |
| TMR | Turkmen |
| VL | Tuvaluan |
| TW | Twi |
| TZE | Tzeltal |
| TZO | Tzotzil |
| UG | Uighur (Cyrillic) |
| K | Ukrainian |
| UB | Umbundu |
| UD | Urdu |
| UR | Urhobo |
| DR | Uruund |
| UZ | Uzbek |
| VLC | Valencian |
| VE | Venda |
| LSV | Venezuelan Sign Language |
| VZ | Vezo |
| VT | Vietnamese |
| WA | Wallisian |
| SA | Waray-Waray |
| W | Welsh |
| XO | Xhosa |
| BM | Yemba |
| YR | Yoruba |
| ZAS | Zambian Sign Language |
| ZSL | Zimbabwe Sign Language |
| ZU | Zulu |
