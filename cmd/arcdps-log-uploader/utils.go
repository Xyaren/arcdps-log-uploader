package main

import (
	"fmt"
	"github.com/lxn/walk"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"runtime"
)

func openBrowser(url string) {
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

func setupLogging() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true, PadLevelText: true})
}

func copyToClipboard(text string) {
	if err := walk.Clipboard().SetText(text); err != nil {
		log.Print("Copy: ", err)
	}
}
