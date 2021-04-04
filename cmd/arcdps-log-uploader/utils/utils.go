package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/lxn/walk"
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
	return versionInfo.StringFileInfo.ProductVersion
}
