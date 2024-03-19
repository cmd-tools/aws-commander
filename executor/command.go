package executor

import (
	"fmt"
	"os/exec"

	"github.com/cmd-tools/aws-commander/logger"
)

func Exec(command []string) string {

	out, err := exec.Command(command[0], command[1:]...).CombinedOutput()

	if err != nil {
		logger.Logger.Err(err).Msg(fmt.Sprintf("Failed to run Exec: %s", out))
	}

	return string(out)
}

func ExecCommand(command string, args []string) string {

	out, err := exec.Command(command, args...).CombinedOutput()

	if err != nil {
		logger.Logger.Err(err).Msg(fmt.Sprintf("Failed to run ExecCommand: %s", out))
	}

	return string(out)
}
