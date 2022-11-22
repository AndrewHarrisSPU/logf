# logf

## What's where

| file | stuff |
| -- | -- |
|`alias.go`| aliases to slog stuff, as well as borrowed std lib code |
|`attrs.go`| procuring and munging attrs |
|`config.go`| configuration, from `New` |
|`handler.go`| Handler |
|`logger.go`| Logger |
|`interpolate.go`| splicer interpolation routines |
|`splicer.go`| splicer lifecycle and writing routines |
|`tty.go`| the TTY device |
|`demo`| `go run`-able TTY demos |
|`testlog`| testing gadgets |

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

A `.` is used to join `Attr` keys in the case of a `Group`-valued `Attr`.

### Examples:
Unkeyed arguments draw from arguments, like `print`:
```
log.Msg("{}", "a")
	-> a

log.Msg("{}, {}", 0, 1)
	-> 0, 1
```

If an unkeyed argument is an `Attr`, it will export:
```
exported := slog.Bool( "exported", true)

log.Msg("{}", exported)
	->  true exported=true

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

### Special verbs
Time and duraton may accept some special verbs:
- a time value may format with `{:RFC3339}`, `{:kitchen}`, `{:stamp}`, or `{:epoch}` (for seconds into the current Unix epoch).
- interpolation can almost accept time layout strings - any occurence of a `:` should be replaced by a `;`.
- a duration value may format with `{:epoch}`. Otherwise, it formats like a string.

### Escaping

Because '{', '}', and ':' are recognized as interpolation tokens, they require escaping to appear in messages passed to logging calls.
A '\\' escapes any following character.

```
log.Msg( "About that struct\\{\\}..." )
	-> msg="About that struct{}...""

log.With(":color", "mauve" ).Msg("The color is {\\:color}.")
	-> msg="The color is mauve."

// Backquotes might be cleaner
Log.With( "x:y ratio", 2 ).Msg( `What a funny ratio: {x\:y ratio}!` )
	-> msg="What a funny ratio: 2!"
```

### Scope and composition
`logf.Handler` doesn't track nested scoping. This is a useful property when interpolating.
Consider:

```
h := slog.NewTextHandler(os.Stderr)
h.WithScope("outer").With("x", 1)

... at considerable distance ...

log := logf.New( Using.Handler(h))
log = log.WithScope("inner").With("x",2)
```

## Etc:

- The ergonomics of in-situ string interpolation, where the interpolation target is named inside of a string, is explored in other languages (to my mind Python's f' strings might be the most compelling example).

There are proposals for Go: https://github.com/golang/go/issues/34174, https://github.com/golang/go/issues/50554. This package doesn't capture variable names as interpolation targets, and it doesn't explore precompiling interpolation strings.

What would change with some hypothetical language-level gadgetry? Matching arguments from logger or context scope to keyed interpolations is a runtime operation. Generating an interpolation dictionary could be a compile time operation. One notable thing might be, at compile time, limiting key strings to valid variable names - escaping around this requires some runtime work.

- `sync.Pool` objects don't shrink in capacity; the mem pinning behavior is simple and workable. This is sort of a general question with `sync.Pool`. This package uses pooled `splicers`; they can pin more than they use, they can't grow very large.

### Performance:
- Low count of allocation calls is possible. Pooling interpolation dictionaries means not allocating a new map, just reusing an empty one. (Notably, Go map capacity doesn't shrink when elements are deleted.)
- While the count of allocations is low, the size of allcations is relatively large.
- A message string must be read to discover interpolation sites, causing some overhead that is always payed. In benchmarks, this is about 1/10th of the cost of the cost of logging a record.
- Each interpolation site adds to the cost of handling a `Record`. Each interpolation is about 1/5th the cost of a `Record` with no interpolation.