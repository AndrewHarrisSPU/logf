/*
Package logf is a logging package extending [slog].

Included are:
  - terminal output tuned for human readability (and fairly configurable)
  - quick/consolidated configuration
  - string interpolation/formatting methods, including error wrapping
  - a smörgåsbord of little ideas that support these things (e.g., lazy Stores, JSON -> JSONValue)

# Hello, world

	package main

	import "github.com/AndrewHarrisSPU/logf"

	func main() {
		log := logf.New().Logger()
		log.Info("Hello, Roswell")
	}

# Interpolation

Generating output similar to the earlier Hello, world progam:

	log = log.With("place", "Roswell")
	log.Infof("Hello, {place}")

Reporting a UFO sighting:

	ufo := errors.New("🛸 spotted")
	log.Errorf("{place}", ufo)

Generating a wrapped error:

	ufo := errors.New("🛸 spotted")
	err := log.WrapErr("{place}", errors.New("🛸 spotted"))

# TTY

The [TTY] component is a [Handler] designed for logging to human eyes.
It pretty-prints lines like:

	▎ 15:04:05 message   key:value

Various layout and formatting details are configurable.

A [TTY] can display tags set with [Logger.Tag] or detected by configuration ([Config.Tag] or [Config.TagEncode]).
Tags can be alternative or auxilliary to long strings of attributes.

# Integration with [slog]

The logf-native [Logger] and [Handler] resemble [slog] counterparts.
A [logf.Logger] can be built from a [slog.Handler], and a [logf.Handler] is a valid [slog.Handler].

Example usage:

Construct a [Logger], which is in

	log := logf.New().Logger()

The resulting logger is based on a [TTY] if standard output is a terminal.
Otherwise, the logger is based on a [slog.JSONHandler].

Passing a [TTY]:

	tty := logf.New().TTY()
	slog.New(tty)

Construct a [Logger], given a [slog.Handler]:

	log := logf.UsingHandler(h)

The resulting logger may be unable interpolate over any attrbiutes set on a non-logf-Handler.
In general, effort is made via type assertions to recover logf types, but recovery isn't always possible.

# testlog

A package [testlog] is also included in logf.
It's offered more in the sense of "this is possible" rather than "this should be used".

# Examples

Note: in examples, some boilereplate is used to trim example output:

	log := log.New().
		Colors(false).		// turns off colors
		ForceTTY(true).		// forces TTY output, even though Example output doesn't write to a terminal
		Printer()		// elides some slog Record fields
*/
package logf
