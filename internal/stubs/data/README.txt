Precompiled gzip-compressed extractors are version-controlled and included in
Go module releases, so `go install github.com/gzj/selfext@latest` works directly.
Maintainers: run `make stubs` after runtime/dependency changes and commit all
updated .gz files. Ordinary builds, installs, and clean must preserve them.
