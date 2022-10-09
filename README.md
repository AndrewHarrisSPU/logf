# logf
Structured logging with string interpolation in Go

## goals
- Explore `x/exp/slog`
- Structured logging is motivated by machine-parsable logging, and optimizes for machine readability. It's a good thing.
- Sometimes developers want a simpler API, with formatting. String interpolation is an experiment in sugar. 
- This isn't zero-allocating. Low allocation is a goal, but there are tradeoffs.

# What is where

| file | stuff |
| -- | -- |
|`alias.go`| aliases to slog stuff, as well as borrowed std lib code |
|`context.go`| CtxLogger |
|`handler.go`| Handler |
|`logger.go`| Logger |
|`minimal.go`| a minimal encoder|
|`splicer.go`| interpolation state |
|`testutil.go`| testing gadgets |
|`text.go`| interpolation buffer ops|
|`using.go`| configuration via Options|

## Opinions That May be Wrong

Part of experimenting with `slog` is figuring out what the opinions are, and what different opinions are possible, and what the implications are. So, `logf` is trying to do some things differently just for the sake of experimenting.

- String interpolation is worth some allocation.
- Munging of small collections, with `Segment`
- Rather than many levels, store level in the `Logger`.
- Contexts store `[]Attr` segments. Contexts are either persistent, and handled with `With`, or transient, and handled by a `CtxLogger`.
- Configuration uses `Using.X` struct.

## Interpolation

### Interpolation symbols
During interpolation, an input message is scanned for interpolation symbols. There are two flavors of these:
- unkeyed `{}`
- keyed `{keystring}`, where an interpolation dictionary is used to find an Attr associated with `keystring`

Both flavors may accomodate a formatting verb, e.g:
```
{:%s} - unkeyed, formatted as a string
{pi:%3.2f} - keyed, formatting the interpolated value as a float as with `fmt` package
```

### Unkeyed arguments

Unkeyed interpolation symbols draw one argument from logging call arguments. One argument is taken per unkeyed symbol:
```
log.Msg("{}", "a")
	-> msg="a"
log.Msg("{}, {}", 0, 1)
	-> msg="0, 1"
```

If an unkeyed interpolation sees an Attr, the Attr is exported. It is not added to the interpolation dictionary, however.
```
exported := slog.Bool( "exported", true)
log.Msg("{}", exported)
	-> msg="true" exported=true
log.Msg("{} {exported}", exported)
	-> msg="true !missing-key" exported=true
```

### Keyed arguments
Any arguments present after exhausting unkeyed interpolations are converted as key-value pairs.
```
Msg("Hi", "name", "Mulder")
	-> msg="Hi" name=Mulder
```

### Escaping

Keys with '{', '}', or ':' cause problems. With some hypothetical language-level f-strings, these might be invalid, as they can't be present in variable 
names. Currently: pretty open question, trying escaping schemes at the time I'm writing this.

## Problems:
- When freeing a splicer, map length rather than map capacity is used to determine if the splicer should be returned to the pool. This may not be ideal.