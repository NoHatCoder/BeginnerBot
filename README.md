# Installation

Install Golang.

Place the contents of this package in a folder that is neither in the GOROOT or GOPATH.

Install the `golang.org/x/net/context` package.

# Programs

There are several commands included under the `cmd` directory. All commands accept `-help` to list flags, but are otherwise minimally documented at present. Note that many flags are left over from Taktician and currently have no effect.

## cmd/playtak

A simple interface to play tak on the command line. To play black run:

```
go run main.go
```

To play white run:

```
go run main.go -white=human -black=minimax:5
```

## cmd/analyzetak

A program that reads PTN files and performs AI analysis on the terminal position.

```
analyzetak FILE.ptn
```

## cmd/taklogger

A bot that connects to playtak.com and logs all games it sees in PTN format.

## cmd/taktician

The AI driver for playtak.com.

Compile with:

```
go build
```

Can be used via:

```
taktician -user USERNAME -pass PASSWORD
```

[tak]: http://cheapass.com/node/215
