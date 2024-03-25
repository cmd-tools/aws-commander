package profile

import (
	"fmt"
	"github.com/cmd-tools/aws-commander/logger"
	"sort"
	"strings"
	"sync"

	"github.com/cmd-tools/aws-commander/executor"
)

type SSO struct {
	Region    string
	StartURL  string
	RoleName  string
	AccountId string
}

type Profile struct {
	Name   string
	Region string
	SSO    SSO
}

type Profiles []Profile

func GetList() Profiles {
	command := "aws"
	args := []string{"configure", "list-profiles"}
	out := executor.ExecCommand(command, args)
	profileNames := strings.Fields(out)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var profiles Profiles
	properties := []string{
		"region",
		"sso_region",
		"sso_start_url",
		"sso_role_name",
		"sso_account_id",
	}

	propertyCount := len(properties)

	for _, profileName := range profileNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			var region, ssoRegion, ssoStartURL, ssoRoleName, ssoAccountId string

			ch := make(chan string, propertyCount)

			// Launch goroutines to fetch profile details concurrently, aws command it's really slow, let's parallelize
			for i, property := range properties {
				logger.Logger.Debug().Msg(fmt.Sprintf("[Worker] Fetching property: %s for profile: %s", property, name))

				go getProfileDetailsByProperty(name, property, ch)
				result := <-ch
				switch {
				case i == 0:
					region = result
				case i == 1:
					ssoRegion = result
				case i == 2:
					ssoStartURL = result
				case i == 3:
					ssoRoleName = result
				case i == 4:
					ssoAccountId = result
				}
			}

			mu.Lock()
			defer mu.Unlock()

			profiles = append(profiles, Profile{
				Name:   name,
				Region: region,
				SSO: SSO{
					Region:    ssoRegion,
					StartURL:  ssoStartURL,
					RoleName:  ssoRoleName,
					AccountId: ssoAccountId,
				},
			})
		}(profileName)
	}

	wg.Wait()

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	return profiles
}

func (profiles Profiles) AsMatrix() [][]string {
	var matrix [][]string

	for _, profile := range profiles {
		matrix = append(matrix, []string{
			profile.Name,
			profile.Region,
			profile.SSO.Region,
			profile.SSO.RoleName,
			profile.SSO.AccountId,
			profile.SSO.StartURL,
		})
	}

	return matrix
}

func getProfileDetailsByProperty(profileName string, property string, ch chan<- string) {
	command := "aws"
	args := []string{"configure", "get", property, "--profile", profileName}
	out := executor.ExecCommand(command, args)
	if len(strings.Fields(out)) == 0 {
		ch <- "n/a"
		return
	}
	ch <- strings.Fields(out)[0]
}
