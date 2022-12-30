# logf

[![Go Reference](https://pkg.go.dev/badge/github.com/AndrewHarrisSPU/logf.svg)](https://pkg.go.dev/github.com/AndrewHarrisSPU/logf)

## Current status

Core functionality seems complete.

API is still being explored. Particularly:

- Some nuances with expanding namespacing, Groups, and LogValuers are subtle - the behavior needs to feel right, is evolving.
- Context and slog integration should be straightforward but hasn't been expiremented with much.

## What's where

| file | stuff |
| -- | -- |
|`alias.go`| aliases to slog stuff, as well as borrowed std lib code |
|`attrs.go`| procuring and munging attrs |
|`config.go`| configuration, from `New` |
|`encoder.go`| TTY encoding logic |
|`fmt.go`| package-level formatting functions |
|`handler.go`| Handler |
|`interpolate.go`| splicer interpolation routines |
|`logger.go`| Logger |
|`splicer.go`| splicer lifecycle and writing routines |
|`styles.go`| TTY styling gadgets |
|`tty.go`| the TTY device |
|`demo`| `go run`-able TTY demos |
|`testlog`| testing gadgets |
