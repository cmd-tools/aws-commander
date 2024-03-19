package helpers

import (
	"strings"

	"github.com/cmd-tools/aws-commander/executor"
)

func GetAWSClientVersion() string {
	cmd := executor.ExecCommand("aws", []string{"--version"})
	parts := strings.Fields(cmd)

	return strings.Split(parts[0], "aws-cli/")[1]
}
