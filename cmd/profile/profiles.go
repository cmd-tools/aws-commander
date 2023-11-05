package profile

import (
	"strings"

	"github.com/cmd-tools/aws-commander/executor"
)

func GetList() []string {
	command := "aws"
	args := []string{"configure", "list-profiles"}
	out := executor.ExecCommand(command, args)
	return strings.Fields(out)
}
