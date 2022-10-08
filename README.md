# logf
Structured logging with string interpolation in Go

## goals
- Explore `x/exp/slog`

Structured logging is motivated by machine-parsable logging, and optimizes for machine readability.

Sometimes developers want a simpler API, with formatting.


##

- level constants in CAPS: slog.INFO, slog.DEBUG rather than slog.InfoLevel, slog.DebugLevel
- 



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

## Goals
- Exploring the slog package
- An API that supports string interpolation and formatting on top of structured logging (and not the other way around)
- A minimal logging API - reducing the line noise and documentation size
- An API that is optionally configured for more human-readable output

## Opinions That May be Wrong

- String interpolation is worth an allocation or two.
- Munging of small collections of information works differently
- Contexts store `[]Attr` segments. Contexts are either persistent, and handled with `With`, or transient, and handled by a `CtxLogger`.
- Configuration uses `Using.X` struct.

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

# Unsolved problems
- Keys with '{', '}', or ':' cause problems. With some hypothetical language-level f-strings, these might be invalid, as they can't be present in variable names.
- When freeing a splicer, map length rather than map capacity is used to determine if the splicer should be returned to the pool. This may not be ideal.