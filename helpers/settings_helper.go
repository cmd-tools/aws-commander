package helpers

import (
	"fmt"
	"github.com/cmd-tools/aws-commander/constants"
	"github.com/cmd-tools/aws-commander/logger"
	"os"
)

func SetSelectedProfile(value string) {
	env := constants.AWS_COMMANDER_SELECTED_PROFILE
	err := os.Setenv(env, value)
	if err != nil {
		panic(err)
	}
	logger.Logger.Debug().Msg(fmt.Sprintf("Saving %s=%s", env, value))
}

func GetSelectedProfile() string {
	env := constants.AWS_COMMANDER_SELECTED_PROFILE
	value := os.Getenv(env)
	logger.Logger.Debug().Msg(fmt.Sprintf("Getting %s=%s", env, value))
	return value
}

func SetSelectedResource(value string) {
	env := constants.AWS_COMMANDER_SELECTED_RESOURCE
	err := os.Setenv(env, value)
	if err != nil {
		panic(err)
	}
	logger.Logger.Debug().Msg(fmt.Sprintf("Saving %s=%s", env, value))
}

func GetSelectedResource() string {
	env := constants.AWS_COMMANDER_SELECTED_RESOURCE
	value := os.Getenv(env)
	logger.Logger.Debug().Msg(fmt.Sprintf("Getting %s=%s", env, value))
	return value
}

func SetSelectedCommand(value string) {
	env := constants.AWS_COMMANDER_SELECTED_COMMAND
	err := os.Setenv(env, value)
	if err != nil {
		panic(err)
	}
	logger.Logger.Debug().Msg(fmt.Sprintf("Saving %s=%s", env, value))
}

func GetSelectedCommand() string {
	env := constants.AWS_COMMANDER_SELECTED_COMMAND
	value := os.Getenv(env)
	logger.Logger.Debug().Msg(fmt.Sprintf("Getting %s=%s", env, value))
	return value
}
