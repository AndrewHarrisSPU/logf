package main

import (
	"github.com/AndrewHarrisSPU/logf"
)

func main() {
	log := logf.New().
		ReplaceAttr(func(a logf.Attr) logf.Attr {
			if a.Key == "secret" {
				return logf.KV("secret", "redacted")
			}
			return a
		}).
		JSON()

	log = log.With("secret", 1)

	log.Msg("{secret}", "secret", 2)

	log.Msg("{group.secret}, {group.group2.secret}", logf.Group("group",
		logf.KV("secret", 3),
		logf.Group("group2",
			logf.KV("secret", 4),
			logf.KV("secret", 5),
		),
	))
}
