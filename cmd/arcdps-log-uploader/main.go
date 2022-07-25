//go:generate goversioninfo -gofile=utils/versioninfo.go -gofilepackage=utils ./_res/versioninfo.json
//go:build (amd64 && 386) || windows

package main

import (
	"github.com/lxn/walk"
	log "github.com/sirupsen/logrus"
	"github.com/xyaren/arcdps-log-uploader/cmd/arcdps-log-uploader/model"
	"github.com/xyaren/arcdps-log-uploader/cmd/arcdps-log-uploader/ui"
	"github.com/xyaren/arcdps-log-uploader/cmd/arcdps-log-uploader/utils"
	"golang.org/x/sys/windows"
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
	var sid *windows.SID

	// Although this looks scary, it is directly copied from the
	// official windows documentation. The Go API for this is a
	// direct wrap around the official C++ API.
	// See https://docs.microsoft.com/en-us/windows/desktop/api/securitybaseapi/nf-securitybaseapi-checktokenmembership
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		log.Fatalf("SID Error: %s", err)
		return false
	}

	token := windows.Token(0)
	isAdmin, _ := token.IsMember(sid)
	return token.IsElevated() || isAdmin
}
