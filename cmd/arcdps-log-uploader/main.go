//go:generate goversioninfo ./_res/versioninfo.json

package main

import (
	log "github.com/sirupsen/logrus"
)

func main() {
	setupLogging()
	log.Info("Starting")

	err := startUi()
	if err != nil {
		panic(err)
	}
	log.Info("Bye")
}
