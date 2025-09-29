# schh

`schh` is a small helper that manages SSH sessions inside GNU Screen. You can add host aliases, start or resume sessions, and keep track of the last session you used for each host.

## Features

- Store SSH targets under short host names.
- Start detached screen sessions that wrap `ssh`.
- Reattach existing sessions or create a new one with an interactive prompt.
- Remember the last session label for quick reconnects.

## Requirements

- Go 1.20 or later
- GNU Screen available on your `PATH`
- SSH client available on your `PATH`

## Build and Install

```sh
make build          # build ./bin/schh
make install        # install into your Go bin directory
```

To keep dependencies tidy and ensure formatting, you can run:

```sh
make tidy
make fmt
```

## Usage

Configure a host alias:

```sh
schh host add prod example.com
```

List configured hosts:

```sh
schh host list
```

Start or attach to sessions:

```sh
schh prod            # interactive prompt
schh prod api        # use an explicit session label
schh --list prod     # show running sessions
schh --last prod     # reconnect to the most recent session
```

Host and session metadata is stored under `~/.config/schh/`.
