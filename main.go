package main

import (
	"os"

	c "github.com/gookit/color"
	"github.com/katbyte/ghp-sync/cli"
	"github.com/katbyte/go-kt/clog"
)

const cmdName = "ghp-sync"

func main() {
	// the log level comes from GHP_SYNC_LOG; read it once here, before anything logs
	clog.SetLevelFromEnv("GHP_SYNC_LOG")

	cmd, err := cli.Make(cmdName)
	if err != nil {
		clog.Log.Errorf("%s", c.Sprintf("<red>%s: building cmd</> %v", cmdName, err))

		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		clog.Log.Errorf("%s", c.Sprintf("<red>%s:</> %v", cmdName, err))

		os.Exit(1)
	}

	os.Exit(0)
}
