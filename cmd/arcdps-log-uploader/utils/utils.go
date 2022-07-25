package utils

import (
	"fmt"
	"github.com/josephspurrier/goversioninfo"
	"github.com/lxn/walk"
	"os"
	"os/exec"
	"runtime"

	log "github.com/sirupsen/logrus"
)

func OpenBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Error(err)
	}
}

func SetupLogging() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true, PadLevelText: true})
}

func CopyToClipboard(text string) {
	if err := walk.Clipboard().SetText(text); err != nil {
		log.Print("Copy: ", err)
	}
}

func Version() string {
	info := VersionInfo()
	return info.StringFileInfo.ProductVersion
}

func VersionInfo() goversioninfo.VersionInfo {
	return versionInfo
}
