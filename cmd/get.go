package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/cli/go-gh"
	"github.com/spf13/cobra"
)

type SSHKey struct {
	Key       string
	Id        int
	Url       string
	Title     string
	CreatedAt time.Time `json:"created_at"`
	Verified  bool
	ReadOnly  bool `json:"read_only"`
}

const randomStringLength = 20

var org string
var repo string
var sshKeyName string
var outputFile string

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
		}

		sshKey, shhKeyError := getSSHKey(sshKeyName)

		if shhKeyError != nil {
			fmt.Println(shhKeyError)
			return
		}

		sessionToken := randomString(randomStringLength)

		fmt.Printf("Requesting certificate to %s/%s\n", org, repo)
		err := requestCertificateCreation(org, repo, sshKey, sessionToken)

		if err != nil {
			fmt.Println(err)
			return
		}

		for counter := 0; counter < 10; counter++ {
			cert, error := fetchCertificate(sessionToken)
			if error == nil {
				fmt.Printf("Certificate is %s\n", cert)
				file, err := os.Create(outputFile)
				if err != nil {
					log.Fatalf("Got error while opening the output file. Err: %s", err.Error())
					return
				}
				writer := bufio.NewWriter(file)
				_, err = writer.WriteString(cert)
				if err != nil {
					log.Fatalf("Got error while writing to a file. Err: %s", err.Error())
				}
				writer.Flush()
				break
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().StringVarP(&org, "org", "o", "", "Organization to use as a certificate authority")
	getCmd.Flags().StringVarP(&repo, "repo", "r", "", "Repo to use as a certificate authority")
	getCmd.Flags().StringVarP(&sshKeyName, "key", "k", "", "SSH key to certify")
	getCmd.Flags().StringVarP(&outputFile, "file", "f", "", "Output file to write the certificate to")
	getCmd.MarkFlagRequired("file")
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func requestCertificateCreation(org string, repo string, sshKey SSHKey, sessionToken string) error {
	client, clientError := gh.RESTClient(nil)
	if clientError != nil {
		return clientError
	}

	fmt.Printf("Key is %s\n", sshKey.Key)

	response := struct{ Message string }{}
	body := fmt.Sprintf(`{"event_type": "certificate-request", "client_payload": {"sessionToken": "%s", "key": "%s", "title": "%s"}}`, sessionToken, sshKey.Key, sshKey.Title)
	bodyReader := bytes.NewReader([]byte(body))
	postError := client.Post(fmt.Sprintf("repos/%s/%s/dispatches", org, repo), bodyReader, &response)

	if postError != nil {
		return postError
	}

	return nil
}

/*
 * Retrieve the list of SSH keys from the current user's GitHub profile
 */
func getSSHKeys() ([]SSHKey, error) {
	client, clientError := gh.RESTClient(nil)
	if clientError != nil {
		return nil, clientError
	}
	response := []SSHKey{}
	postError := client.Get("user/keys", &response)

	if postError != nil {
		return nil, postError
	}
	return response, nil
}

/*
 * Retrieve a single SSH key from the current user's GitHub profile
 */
func getSSHKey(sshKeyName string) (SSHKey, error) {
	fmt.Println("Getting ssh keys as none was provided")
	keys, error := getSSHKeys()

	if error != nil {
		return SSHKey{}, error
	}

	if sshKeyName == "" {
		if len(keys) == 0 {
			return SSHKey{}, fmt.Errorf("no SSH keys found on your GitHub profile. Please add one")
		} else if len(keys) > 1 {
			return SSHKey{}, fmt.Errorf("%d SSH key were found on your GitHub profile. Use the --key flag to specify which one you want to use", len(keys))
		} else {
			return keys[0], nil
		}
	} else {
		for _, key := range keys {
			if key.Title == sshKeyName {
				return key, nil
			}
		}
	}
	return SSHKey{}, fmt.Errorf("no SSH key with name %s was found on your GitHub profile. Please add one", sshKeyName)
}

func fetchCertificate(sessionToken string) (string, error) {
	client, clientError := gh.RESTClient(nil)
	if clientError != nil {
		return "", clientError
	}

	fmt.Printf("Fetching certificate\n")

	response := struct {
		Message     string
		Certificate string
	}{}
	body := fmt.Sprintf(`{"sessionToken": "%s"}`, sessionToken)
	bodyReader := bytes.NewReader([]byte(body))
	postError := client.Post("https://ssh-cert-app.ngrok.io/ssh-cert-app/fetch", bodyReader, &response)

	if postError != nil {
		return "", postError
	}

	return response.Certificate, nil
}
