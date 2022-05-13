# wf

[![godoc](https://pkg.go.dev/badge/github.com/bow/wf)](https://pkg.go.dev/github.com/bow/wf)
[![ci](https://github.com/bow/wf/actions/workflows/ci.yml/badge.svg)](https://github.com/bow/wf/actions?query=branch%3Amaster)
[![coverage](https://api.codeclimate.com/v1/badges/d6472f5c514e9ade0c3a/test_coverage)](https://codeclimate.com/github/bow/wf/test_coverage)


wf waits until TCP server(s) are ready to accept connections.


## Why?

The main use case for wf is to make containerized applications that depend
on external services more robust by waiting for those actual services to be
ready, prior to application start. wf can wait on multiple TCP servers and
is provided as a single, static binary for linux/amd64, so it can be added
into the container and used as-is.


## Usage

wf is provided as a command line application.

    $ wf --help

    Wait until TCP server(s) are ready to accept connections

    Usage:
      wf [FLAGS] ADDRESS...

    Flags:
      -t, --timeout duration     set wait timeout (default 5s)
      -f, --poll-freq duration   set connection poll frequency (default 500ms)
      -q, --quiet                suppress waiting messages
      -h, --help                 help for wf
          --version              version for wf

The functionalities themselves are provided as a Go library in the
[wait](https://godoc.org/github.com/bow/wf/wait) package. Refer to the
relevant GoDoc documentation for a complete documentation.


## Development

wf was developed using Go 1.18 on `linux/amd64`. Other versions and/or
platforms may work but have not been tested.

    # Clone the repository.
    $ git clone https://github.com/bow/wf

    # Install the development dependencies.
    $ make install-dev

    # Run the tests and/or linter.
    $ make test
    $ make lint

    # Build the wf binary.
    $ make bin

    # When in doubt, just run `make`.
    $ make


## Credits

wf was inspired by the [wait-for-it.sh](https://github.com/vishnubob/wait-for-it) script by
@vishnubob.
