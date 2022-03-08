package cmd

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh"
	"github.com/spf13/cobra"
)

var org string
var repo string

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a new SSH certificate from GitHub",
	Run: func(cmd *cobra.Command, args []string) {
		if org == "" && repo == "" {
			// Not flag was set, let's use the current repo if we are actually in a repo
			currentRepoObj, _ := gh.CurrentRepository()
			if currentRepoObj != nil {
				org = currentRepoObj.Owner()
				repo = currentRepoObj.Name()
			}
		}

		missingFlagNames := []string{}
		if org == "" {
			missingFlagNames = append(missingFlagNames, "org")
		}
		if repo == "" {
			missingFlagNames = append(missingFlagNames, "repo")
		}
		if len(missingFlagNames) > 0 {
			fmt.Printf("required flag(s) \"%s\" not set \n", strings.Join(missingFlagNames, `", "`))
			return
		} else {
			fmt.Printf("Requesting certificate to %s/%s\n", org, repo)
			requestCertificateCreation(org, repo)
		}
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().StringVarP(&org, "org", "o", "", "Organization to use as a certificate authority")
	getCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repo to use as a certificate authority")
}

func requestCertificateCreation(org string, repo string) error {
	key := "xxxxxx"
	repoDisptachAPICallArgs := []string{"api", "-X", "POST", fmt.Sprintf("/repos/%s/%s/dispatches", org, repo), "-f", fmt.Sprintf("event_type=certificate-request client_payload={\"key\":\"%s\"}", key)}

	_, _, err := gh.Exec(repoDisptachAPICallArgs...)

	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
