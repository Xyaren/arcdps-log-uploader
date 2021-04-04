//go:generate goversioninfo -gofile=utils/versioninfo.go -gofilepackage=utils ./_res/versioninfo.json

package main

import (
	"os"

	"github.com/lxn/walk"
	log "github.com/sirupsen/logrus"
	"github.com/xyaren/arcdps-log-uploader/cmd/arcdps-log-uploader/model"
	"github.com/xyaren/arcdps-log-uploader/cmd/arcdps-log-uploader/ui"
	"github.com/xyaren/arcdps-log-uploader/cmd/arcdps-log-uploader/utils"
)

func main() {
	utils.SetupLogging()
	log.Info("Starting")

	if runningWithAdminPrivileges() {
		walk.MsgBox(nil, "Administrator mode is not supported",
			"Due to windows security constraints, drag & drop "+
				"is not supported for applications running in administrative mode.",
			walk.MsgBoxOK|walk.MsgBoxIconWarning|walk.MsgBoxTaskModal)
	} else {
		start()
	}
	log.Info("Bye")
}

func start() {
	model.StartWorkerGroup()

	var err = ui.StartUI()
	if err != nil {
		panic(err)
	}

	model.CloseQueue()
}

func runningWithAdminPrivileges() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}
