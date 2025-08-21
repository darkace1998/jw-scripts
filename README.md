# JW Scripts (Go Version)

*These methods of acessing jw.org are, while legal, not officially supported by the organisation. Use them if you find it worth the time, pain and risk. But first, please take the time to read [w18.04 30-31](https://wol.jw.org/en/wol/d/r1/lp-e/2018364). Then consider buing a device which has official support for JW Broadcasting app. Like a Roku, Apple TV or Amazon Fire TV. It will give you a better and safer experience.*

### JW Broadcasting and Publications anywhere

With these scripts you can get the latest JW Broadcasting videos and publications automatically downloaded. You can turn a computer (like a Raspberry Pi) into a JW TV, either by streaming directly, or by playing downloaded videos from your collection.

**Note**: The publication download feature (`jwb-books`) is currently a framework implementation. The JW.org API does not currently provide access to publications/books - only broadcasting content (videos and audio). See [docs/BOOK_DOWNLOAD_ANALYSIS.md](docs/BOOK_DOWNLOAD_ANALYSIS.md) for details.

## Get started

This project is now written in Go. You have two options to get started:

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

To build the `jwb-index` and `jwb-offline` executables, run the following command from the root of the project:

```bash
go build -o bin/ ./...
```

This will create the executables in a `bin` directory.

### Running the applications

#### JW Broadcasting Videos and Audio

For example, to download the latest videos in Swedish, you would run:

```bash
./bin/jwb-index --download --latest --lang=Z
```

To play downloaded videos, you can use the `jwb-offline` command:

```bash
./bin/jwb-offline /path/to/your/videos
```

#### JW Publications (Books, Magazines) - Framework Implementation

**Note**: This feature is currently a framework implementation as the JW.org API does not provide access to publications.

```bash
# Check API status and limitations
./bin/jwb-books

# List available book categories (when API becomes available)
./bin/jwb-books --list-categories

# Download books by category in PDF format (when API becomes available)  
./bin/jwb-books --category=bible-study --format=pdf --output=./books

# Download magazines in EPUB format (when API becomes available)
./bin/jwb-books --category=magazines --format=epub --output=./publications

# Search for specific publications (when API becomes available)
./bin/jwb-books --search="watchtower 2024"
```

See [docs/BOOK_DOWNLOAD_ANALYSIS.md](docs/BOOK_DOWNLOAD_ANALYSIS.md) for detailed information about the API analysis and framework implementation.

Next, check out the [Wiki pages](https://github.com/allejok96/jw-scripts/wiki) for more examples and options. The command-line flags are the same as the original Python version.

## Questions

#### Isn't there an easier way to watch JW Broadcasting in Kodi?

Yes, I'm keeping an add-on alive [here](https://github.com/allejok96/plugin.video.jwb-unofficial).

#### Why is the video download so slow?

~~It seems to be realated to the `--limit-rate` flag ([why?](https://github.com/allejok96/jw-scripts/wiki/How-it-works#batch-downloading)).~~ 

**Fixed!** The rate limiting implementation has been improved to provide smooth downloads at the specified rate limits without the previous throttling issues.

*But please, somebody think of the servers!* :-)

#### What happened to the script for Bible recordings?

Since all recordings can be easily downloaded from the website, and the script couldn't do more than one publication at a time I didn't see any practical use for it.

But checkout [@vbastianpc](https://github.com/vbastianpc)'s nice fork of it that can download sign language publications.

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
