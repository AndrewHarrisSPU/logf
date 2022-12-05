/*
Package logf is a logging package extending [slog].

Often, a structured logging API embeds the expectation that it isn't logging for human eyes.
This is unambiguously a good idea in a lot of situations. Automated processing of log lines is powerful.

Still, logf extends [slog] in the other direction, to be nice for human readers.
It's an interesting problem to work out what the trade-offs are, and how to provide an opt-in API
that allows for varying kinds of functionality here without demanding them or paying for them at runtime.

Work in progress!

# Hello, world

	package main

	import "github.com/AndrewHarrisSPU/logf"

	func main() {
		log := logf.New().Logger()
		log.Msg("Hello, Roswell")
	}

# Interpolation

Generating output similar to the earlier Hello, world progam:

	log = log.With("place", "Roswell")
	log.Msgf("Hello, {place}")

Generating a wrapped error with the error string "Roswell: 🛸 spotted":

	err := log.NewErr("{place}", errors.New("🛸 spotted"))

# TTY

The [TTY] component is a [Handler] designed for logging to human eyes.
It pretty-prints lines like:

	▎ 15:04:05 message   key:value

Various layout and formatting details are configurable.

As an alternative or an auxilliary mode to key:value pairs, a [TTY] also prints tags.

# Tags

A [TTY] can display tags set with [Logger.Tag] or detected by configuration ([Config.Tag] or [Config.TagEncode]).
Tags can be alternative or auxilliary to long strings of attributes.

# Integration with [slog]

The logf-native [Logger] and [Handler] resemble [slog] counterparts.
A [logf.Logger] can be built from a [slog.Handler], and a [logf.Handler] is a valid [slog.Handler].

Example usage:

Construct a [Logger], given a [slog.Handler]:

	log := logf.UsingHandler(h)

Construct a [Logger], given a [context.Context]

	log := logf.FromContext(ctx)

Passing a [TTY]:

	tty := logf.New().TTY()
	slog.New(tty)

# testlog

A package [testlog] is also included in logf.
It's offered more in the sense of "this is possible" rather than "this should be used".

# Examples
*/
package logf
