package executor

import (
	"log"
	"os/exec"
)

func Exec(command []string) string {

	out, err := exec.Command(command[0], command[1:]...).Output()

	if err != nil {
		log.Fatal(err)
	}

	return string(out)
}

func ExecCommand(command string, args []string) string {

	out, err := exec.Command(command, args...).Output()

	if err != nil {
		log.Fatal(err)
	}

	return string(out)
}
