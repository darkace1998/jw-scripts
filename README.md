# JW Scripts (Go Version)

This project is a Go-based reimplementation of the [original Python scripts](https://github.com/allejok96/jw-scripts) for interacting with jw.org content. It offers improved performance and modern features while maintaining compatibility with the original command-line flags.

*These methods of accessing jw.org are, while legal, not officially supported by the organisation. Use them if you find it worth the time, pain and risk. But first, please take the time to read [w18.04 30-31](https://wol.jw.org/en/wol/d/r1/lp-e/2018364). Then consider buying a device which has official support for JW Broadcasting app. Like a Roku, Apple TV or Amazon Fire TV. It will give you a better and safer experience.*

### JW Broadcasting and Publications anywhere

With these scripts you can get the latest JW Broadcasting videos and publications automatically downloaded. You can turn a computer (like a Raspberry Pi) into a JW TV, either by streaming directly, or by playing downloaded videos from your collection.

## Get started

You have two options to get started with the Go version of JW Scripts:

### Option 1: Download Pre-built Binaries (Recommended)

Pre-built binaries are available for multiple platforms from the [Releases page](https://github.com/darkace1998/jw-scripts/releases/latest):

- **Linux** (amd64, arm64)
- **Windows** (amd64, arm64) 
- **macOS** (Intel, Apple Silicon)

Simply download the appropriate binary for your platform and make it executable (Linux/macOS):
```bash
chmod +x jwb-index-linux-amd64 jwb-offline-linux-amd64
```

### Option 2: Building from Source

If you prefer to build from source, you will need to have Go installed on your system.

### Building the project

To build all the command-line tools, run the following command from the root of the project:

```bash
go build -o bin/ ./cmd/...
```

This will create the executables in a `bin` directory.

### Running the applications

#### JW Broadcasting Videos and Audio

For example, to download the latest videos in Swedish, you would run:

```bash
./bin/jwb-index --download --latest --lang=S
```

To play downloaded videos, you can use the `jwb-offline` command:

```bash
./bin/jwb-offline /path/to/your/videos
```

#### JW Music Downloads

The `jwb-music` command is a specialized tool for downloading all music files from jw.org:

```bash
# Download all music files in English
./bin/jwb-music

# Download music files in Spanish with friendly filenames
./bin/jwb-music --lang=S --friendly

# List available music categories
./bin/jwb-music --list-categories

# Download only specific music categories
./bin/jwb-music --category=AudioOriginalSongs,SJJChorus

# Download to a specific directory
./bin/jwb-music ./my-music-folder
```

The command downloads from all music-related categories including:
- Original Songs
- "Sing Out Joyfully" (Meetings, Vocals, Instrumental)  
- Children's Songs
- Kingdom Melodies

#### JW Publications (Books, Magazines) - Framework Implementation

**Note**: This feature is currently a framework implementation as the JW.org API does not provide access to publications.

```bash
# Display help information
./bin/jwb-books --help

# List supported languages
./bin/jwb-books --list-languages

# List available book categories
./bin/jwb-books --list-categories --language E

# List supported download formats
./bin/jwb-books --list-formats

# Search for specific publications
./bin/jwb-books --search="watchtower" --language E

# Download books by category in PDF format
./bin/jwb-books --category=bible-study --language E --format=pdf --output=./books

# Download magazines in EPUB format
./bin/jwb-books --category=magazines --language E --format=epub --output=./publications
```

See [docs/BOOK_DOWNLOAD_ANALYSIS.md](docs/BOOK_DOWNLOAD_ANALYSIS.md) for detailed information about the API analysis and framework implementation.

Next, check out the [Wiki pages](https://github.com/allejok96/jw-scripts/wiki) for more examples and options. The command-line flags are the same as the original Python version.

## Questions

#### Is this legal?

Yes. The [Terms of Service](http://www.jw.org/en/terms-of-use/) allows:

> distribution of free, non-commercial applications designed to download electronic files (for example, EPUB, PDF, MP3, AAC, MOBI, and MP4 files) from public areas of this site.

I've also been in contact with the Scandinavian branch office, and they have confirmed that using software like this is legal according to the ToS.

___

## Development and Contributing

This project uses GitHub Actions for automated testing and releases:

- **Continuous Integration**: Automatic testing on multiple Go versions, linting, and security scanning
- **Automated Releases**: Cross-platform binaries are automatically built and released when tags are pushed
- **Code Quality**: Enforced code formatting, linting, and test coverage

To contribute, see [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

### Creating a Release

To create a new release with pre-built binaries:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This will automatically build binaries for all supported platforms and create a GitHub release.

___

If you have a feature request or have been bitten by a bug, please [create an issue](https://github.com/allejok96/jw-scripts/issues), and I'll see what I can do.
