//go:generate goversioninfo ./_res/versioninfo.json

package main

import (
	"github.com/lxn/walk"
	log "github.com/sirupsen/logrus"
	"os"
)

func main() {
	setupLogging()
	log.Info("Starting")

	if runningWithAdminPrivileges() {
		walk.MsgBox(nil, "Administrator mode is not supported", "Due to windows security constraints, drag & drop is not supported for applications running in administrative mode.", walk.MsgBoxOK|walk.MsgBoxIconWarning|walk.MsgBoxTaskModal)
	} else {
		start()
	}
	log.Info("Bye")
}

func start() {
	var err = startUi()
	if err != nil {
		panic(err)
	}
}

func runningWithAdminPrivileges() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	return true
}
