# attie

AT Protocol Terminal Interface Explorer

## Features

- Browse PDS and repos by handle or DID
- View collections and records

![demo](https://vhs.charm.sh/vhs-7oKRnStqGJrDA7EI9TcmGe.gif)

## Installation

```bash
go install github.com/treethought/attie@latest
```

Or build from source:

```bash
git clone https://github.com/treethought/attie
cd attie
go build
```

## Usage

```
attie
```

Launch with optional handle, DID, or AT URI


View an account's repo
```bash
attie baileytownsend.dev
```

Jump to an account's records of a collection

```
./attie at://did:plc:b2p6rujcgpenbtcjposmjuc3/network.cosmik.collection
```

Jump directly to a record
```
attie at://did:plc:sppiplftd2sxt3hbw7htj3b5/sh.tangled.repo/3meytrdho7p22
```
## Keybindings

- `ctrl+k` - Open command palette
- `esc` - Navigate back
- `enter` - Select item
- `ctrl+c` / `q` - Quit
