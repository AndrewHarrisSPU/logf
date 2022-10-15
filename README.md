# logf
Structured logging with string interpolation in Go

## Goals
- Explore `x/exp/slog`
- Structured logging is motivated by machine-parsable logging, and optimizes for machine readability. It's a good thing. Still, sometimes a small API with formatting is nice to use. `logf` is an experiment in string interpolation sugar.

## What's where

| file | stuff |
| -- | -- |
|`alias.go`| aliases to slog stuff, as well as borrowed std lib code |
|`context.go`| CtxLogger |
|`handler.go`| Handler |
|`logger.go`| Logger |
|`minimal.go`| a minimal encoder|
|`splicer.go`| splicer mgmt, join, matching |
|`splicer2.go`| message scan |
|`splicer3.go`| interpolate and write |
|`testutil.go`| testing gadgets |
|`text.go`| interpolation buffer ops|
|`using.go`| configuration via Options|

## TODO
- Fix source depth
- Benchmarking - time and allocations are possible, but are there other useful metrics?
   Size of pool (how big is a splicer relative to just a byte buffer?)

## Opinions That May be Wrong

Part of experimenting with `slog` is figuring out what the opinions are, and what different opinions are possible, and what the implications are. So, `logf` is trying to do some things differently just for the sake of experimenting.

- String interpolation is worth some allocation.
- Munging of small collections, with `Segment`
- Rather than many levels, store level in the `Logger`.
- Contexts store `[]Attr` segments. Contexts are either persistent, and handled with `With`, or transient, and handled by a `CtxLogger`.
- Configuration uses `Using.X` struct.

## Interpolation

### Interpolation symbols
Message strings in `logf` may contain interpolation symbols. There are two varieties of interpolation symbols:
- unkeyed `{}`, which consume arguments like `fmt` or the built-in `print`
- keyed `{keystring}`, where an interpolation dictionary associates `keystring` with a `slog.Value`. The interpolation dictionary is populated by a `Handler`'s structured `Attr`s, or arguments in a logging call.

Both flavors accomodate formatting verbs (generally, the verb passed to `fmt`):
```
{:%s} - unkeyed, formatted as a string
{pi:%3.2f} - keyed, formatting as a float
```

### Examples:
Unkeyed arguments draw from arguments, like `print`:
```
log.Msg("{}", "a")
	-> msg="a"

log.Msg("{}, {}", 0, 1)
	-> msg="0, 1"
```

If an unkeyed argument is an `Attr`, it will export:
```
exported := slog.Bool( "exported", true)

log.Msg("{}", exported)
	-> msg="true" exported=true

log.Msg("{} {exported}", exported)
	-> msg="true true" exported=true
```

After exhausting unkeyed interpolations, `Attr`s are converted as key-value pairs (like `slog`):
```
log.Msg("Hi", "name", "Mulder")
	-> msg="Hi" name=Mulder

log.Msg("Hi, {name}", "name", "Mulder")
	-> msg="Hi, Mulder" name=Mulder
```

### Groups
A `.` may be used in named interpolation keys to access grouped `Attr`s:
A `Group`-valued `Attr` will expand on interpolation.

```
log = log.With(
	slog.Group( "1",
		slog.String( "i", "first off, this thing" ),
		slog.String( "ii", "and another thing" )))
		
log.Msg("{1.i}")
	-> msg="first off, this thing"
log.MSg("{1.ii}")
	-> msg="and another thing"
log.Msg("{1}")
	-> msg="[i=first off, this thing ii=and another thing]"
```

### Ordering
`Attr`s join the interpolation dictionary in a specific order: `Handler` segments, `context.Context` segments, arguments.
If non-unique `Attr` keys are seen, the last seen `Attr` wins.

### Special verbs
Time and duraton may accept some special verbs:
- a time value may format with `{:RFC3339}`, `{:kitchen}`, `{:stamp}`, or `{:epoch}` (for seconds into the current Unix epoch).
- interpolation can almost accept time layout strings - any occurence of a `:` should be replaced by a `;`.
- a duration value may format with `{:fast}` (like epoch). Otherwise, it formats like a string.

### Escaping

Because '{', '}', and ':' are used as interpolation tokens, they may need to be escaped in messages passed to logging calls.
A '\\' reads as an escape, but will itself need to be escaped in double-quoted strings.

```
log.Msg( "About that struct\\{\\}..." )
	-> msg="About that struct{}...""

log.With(":color", "mauve" ).Msg("The color is {\\:color}.")
	-> msg="The color is mauve."

// Backquotes might be cleaner
Log.With( "x:y ratio", 2 ).Msg( `What a funny ratio: {x\:y ratio}!` )
	-> msg="What a funny ratio: 2!"
```

## Yet unsolved problems:
- Precompiling messages (as a compiler might with language level fstrings) would be a bit more performant, and more straightforward about escaping.
- `sync.Pool` objects don't shrink in size, the mem pinning behavior is simple and workable. (sort of a general question with `sync.Pool`).