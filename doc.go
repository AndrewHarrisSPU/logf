/*
Package logf is a logging package extending [slog].

Often, a structured logging API draws a sharp lines around logging, printing, formatting, and displaying data.
logf supposes that a structured logging device can be well-suited to printing, formatting, and displaying log lines or log-like text.

It's a situationally useful approach. Situations where logf tries to be nice:
  - When logging, printing, and formatting overlap, maintaing just one way of capturing and propagating relevant data is nice. (Structured logging APIs do well at capturing data!)
  - Selectively including some elements of structured data in log messages or wrapped errors is nice.
  - When connecting human eyes to log lines, some presentation is nice.

logf has also been a way to explore [slog], which I think is an excellent idea and addresses a real need around integrating disparte logging infrastructure.

# Hello, world

	package main

	import "github.com/AndrewHarrisSPU/logf"

	func main() {
		log := logf.New().Logger()
		log.Msg("Hello, Roswell")
	}

# Interpolation

Structured logging libraries conventionally emit captured structure by including a list of key-value elements in a log line.
For occasions where other ways of liberating captured structure would be useful, logf offers string interpolation.

Generating output similar to the earlier Hello, world progam:

	log.With("place", "Roswell")
	log.Msg("Hello, {place}")

Generating a wrapped error with the error string "Roswell: 🛸 spotted":

	err := log.Errf("{place}", errors.New("🛸 spotted"))

# TTY

The [TTY] component is a [Handler] designed for logging to human eyes.
It pretty-prints lines like:

	01:23:45   INFO    label  msg: err  key=value

Various layout and formatting details are configurable.

Additionally, a [TTY] can be configured to stream, displaying recent rather than total log output (or some mix thereof).
Short programs in the demo folder best demonstrate this [TTY] functionality.

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

(alternatively, using [logf.StdTTY]):

	slog.New(logf.StdTTY)

# testlog

A package [testlog] is also included in logf.
It's offered more in the sense of "this is possible" rather than "this should be used".

# Examples
*/
package logf
