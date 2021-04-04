package utils

import (
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	"github.com/sirupsen/logrus"
)

const repo = "Xyaren/arcdps-log-uploader"

func DoUpdate(latest *selfupdate.Release) {
	exe, err := os.Executable()
	if err != nil {
		logrus.Println("Could not locate executable path")
		return
	}
	if err := selfupdate.UpdateTo(latest.AssetURL, exe); err != nil {
		logrus.Println("Error occurred while updating binary:", err)
		return
	}
	logrus.Println("Successfully updated to version", latest.Version)
}

func CheckUpdate() (*selfupdate.Release, bool) {
	//goland:noinspection GoBoolExpressions
	version := Version()
	if version == "develop" || version == "snapshot" {
		return nil, true
	}
	v := semver.MustParse(strings.Replace(version, "v", "", 1))
	latest, found, err := selfupdate.DetectLatest(repo)
	if err != nil {
		logrus.Println("Error occurred while detecting version:", err)
		return nil, true
	}

	if !found || latest.Version.LTE(v) {
		logrus.Println("Current version is the latest")
		return nil, true
	}
	return latest, false
}

func ForkExec() error {
	argv0, err := lookPath()
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	p, err := os.StartProcess(argv0, os.Args, &os.ProcAttr{
		Dir:   wd,
		Env:   os.Environ(),
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
		Sys: &syscall.SysProcAttr{
			HideWindow: false,
		},
	})
	if err != nil {
		return err
	}
	logrus.Println("spawned child", p.Pid)
	return nil
}

func lookPath() (argv0 string, err error) {
	argv0, err = exec.LookPath(os.Args[0])
	if err != nil {
		return
	}
	if _, err = os.Stat(argv0); err != nil {
		return
	}
	return
}
