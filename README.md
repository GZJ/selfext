# selfext

**selfext** is a cross-platform tool that turns any compressed file into a self-extracting file.

## Security Notice!!!

- Be cautious about running executable files sent by others.
- Do not run executable files sent by strangers indiscriminately.

## Install

### Binary release

You can download the precompiled binary release for your platform from the releases page:

- https://github.com/gzj/selfext/releases

### Build from Source
To build the `selfext` tool, you need to have [Go](https://golang.org/dl/) installed. Then, run the following commands:

```sh
git clone https://github.com/yourusername/selfext.git
cd selfext
make
```
## Usage

To turn `example.zip` into a self-extracting file, run:

```sh
selfext example.zip
```

This will generate `example.zip.exe`, which extracts `example.zip` into the `example` directory when run.



## How it works

- **Selfext** embeds a specified Golang SDK for different platforms.
- It embeds `assets_generator` and `wrapper_generator` files, along with all required dependency packages.
- Upon first run, it extracts and unpacks the Golang SDK, the wrapper, and its dependencies into the user's configuration directory.
- By reading command-line parameters specifying the target archive, os, and architecture, the tool uses the Go SDK to compile `wrapper_generator` and `assets_generator` outputs along with the specified archive into a platform-specific self-extracting archive.

## License

This project is licensed under the MIT License. See the [LICENSE](https://github.com/GZJ/selfext/blob/master/LICENSE) file for details.
