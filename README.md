# logf

## TODO:

- interpolation of groups: shouldn't need to .LogValue() ...

## What's where

| file | stuff |
| -- | -- |
|`alias.go`| aliases to slog stuff, as well as borrowed std lib code |
|`attrs.go`| procuring and munging attrs |
|`config.go`| configuration, from `New` |
|`encoder.go`| TTY encoding logic |
|`handler.go`| Handler |
|`interpolate.go`| splicer interpolation routines |
|`logger.go`| Logger |
|`splicer.go`| splicer lifecycle and writing routines |
|`styles.go`| TTY styling gadgets |
|`tty.go`| the TTY device |
|`demo`| `go run`-able TTY demos |
|`testlog`| testing gadgets |