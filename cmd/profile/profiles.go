package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cmd-tools/aws-commander/executor"
	"github.com/cmd-tools/aws-commander/logger"
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

func (profiles Profiles) GetProfileNames() []string {
	var list []string

	for _, profile := range profiles {
		list = append(list, profile.Name)
	}

	return list
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

// ssoTokenCache represents the structure of an SSO cache file that contains an access token
type ssoTokenCache struct {
	StartURL    string `json:"startUrl"`
	Region      string `json:"region"`
	AccessToken string `json:"accessToken"`
	ExpiresAt   string `json:"expiresAt"`
}

// listAccountRolesResponse represents the response from aws sso list-account-roles
type listAccountRolesResponse struct {
	RoleList []struct {
		RoleName  string `json:"roleName"`
		AccountId string `json:"accountId"`
	} `json:"roleList"`
}

// FindProfile returns the Profile matching the given name, or nil if not found
func (profiles Profiles) FindProfile(name string) *Profile {
	for i := range profiles {
		if profiles[i].Name == name {
			return &profiles[i]
		}
	}
	return nil
}

// getAccessToken reads the SSO cache directory and returns the access token
// for the given start URL. Returns empty string if no valid token is found.
func getAccessToken(startURL string) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to get home directory")
		return ""
	}

	cacheDir := filepath.Join(homeDir, ".aws", "sso", "cache")
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to read SSO cache directory")
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(cacheDir, entry.Name()))
		if err != nil {
			continue
		}

		var cache ssoTokenCache
		if err := json.Unmarshal(data, &cache); err != nil {
			continue
		}

		// Match by start URL and ensure token exists
		if cache.StartURL != startURL || cache.AccessToken == "" {
			continue
		}

		// Check expiry
		expiresAt, err := time.Parse(time.RFC3339, cache.ExpiresAt)
		if err != nil {
			logger.Logger.Debug().Str("expiresAt", cache.ExpiresAt).Msg("Failed to parse SSO token expiry")
			continue
		}
		if time.Now().After(expiresAt) {
			logger.Logger.Debug().Str("startUrl", startURL).Msg("SSO token expired")
			continue
		}

		return cache.AccessToken
	}

	return ""
}

// ListAccountRoles fetches the list of SSO role names available for a given profile.
// It reads the SSO access token from the cached SSO session and calls
// aws sso list-account-roles. Returns role names or an error message.
func ListAccountRoles(p *Profile) ([]string, error) {
	if p.SSO.AccountId == "" || p.SSO.AccountId == "n/a" {
		return nil, fmt.Errorf("profile %s has no SSO account ID configured", p.Name)
	}
	if p.SSO.StartURL == "" || p.SSO.StartURL == "n/a" {
		return nil, fmt.Errorf("profile %s has no SSO start URL configured", p.Name)
	}

	accessToken := getAccessToken(p.SSO.StartURL)
	if accessToken == "" {
		return nil, fmt.Errorf("no valid SSO session found for %s. Run 'aws sso login --profile %s' first", p.SSO.StartURL, p.Name)
	}

	region := p.SSO.Region
	if region == "" || region == "n/a" {
		region = p.Region
	}

	args := []string{
		"sso", "list-account-roles",
		"--account-id", p.SSO.AccountId,
		"--access-token", accessToken,
		"--region", region,
		"--output", "json",
	}

	out := executor.ExecCommand("aws", args)

	var response listAccountRolesResponse
	if err := json.Unmarshal([]byte(out), &response); err != nil {
		logger.Logger.Error().Err(err).Str("output", out).Msg("Failed to parse list-account-roles response")
		return nil, fmt.Errorf("failed to list roles: %s", strings.TrimSpace(out))
	}

	var roles []string
	for _, role := range response.RoleList {
		roles = append(roles, role.RoleName)
	}

	sort.Strings(roles)
	return roles, nil
}

// UpdateSSORole updates the sso_role_name for a given profile using aws configure set
func UpdateSSORole(profileName string, roleName string) error {
	args := []string{"configure", "set", "sso_role_name", roleName, "--profile", profileName}
	out := executor.ExecCommand("aws", args)
	trimmed := strings.TrimSpace(out)
	if trimmed != "" {
		logger.Logger.Debug().Str("output", trimmed).Msg("aws configure set output")
	}
	return nil
}
