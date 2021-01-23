# wf

[![latest](https://img.shields.io/github/v/release/bow/wf?labelColor=4d4d4d&color=007c5b&style=flat)](https://github.com/bow/wf/releases)
[![doc](https://img.shields.io/badge/godoc-reference-007c5b?style=flat)](https://godoc.org/github.com/bow/wf/wait)
[![ci](https://img.shields.io/travis/bow/wf?labelColor=4d4d4d&color=007c5b&style=flat)](https://travis-ci.org/bow/wf)
[![cov](https://img.shields.io/codeclimate/coverage/bow/wf?labelColor=4d4d4d&color=007c5b&style=flat)](https://codeclimate.com/github/bow/wf)
[![qual](https://img.shields.io/codeclimate/maintainability/bow/wf?labelColor=4d4d4d&color=007c5b&style=flat)](https://codeclimate.com/github/bow/wf)

`wf` waits until TCP server(s) are ready to accept connections.


## Why?

The main use case for `wf` is to make containerized applications that depend
on external services more robust by waiting for those actual services to be
ready, prior to application start. `wf` can wait on multiple TCP servers and
is provided as a single, static binary for linux/amd64, so it can be added
into the container and used as-is.


## Usage

`wf` is provided as a command line application.

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
[wait](https://godoc.org/github.com/bow/wait-for/wait) package. Refer to the
relevant GoDoc documentation for a complete documentation.


## Development

`wf` was developed using Go 1.15 on linux/amd64. Other versions and/or
platforms may work but have not been tested.

    # Clone the repository.
    $ git clone https://github.com/bow/wf

    # Install the development dependencies.
    $ make dev

    # Run the tests and/or linter.
    $ make test
    $ make lint

    # Build the wf binary.
    $ make bin


## Credits

`wf` was inspired by the [wait-for-it.sh](https://github.com/vishnubob/wait-for-it) script by @vishnubob.
