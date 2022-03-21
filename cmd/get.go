package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path"
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
var pubKey string
var serverRootURL string

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a new SSH certificate from GitHub",
	Run: func(cmd *cobra.Command, args []string) {
		if org == "" && repo == "" {
			// No flag was set, let's use the current repo if we are actually in a repo
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

		// Get the SSH pub key
		sshKey, shhKeyError := getSSHKey(pubKey)

		if shhKeyError != nil {
			fmt.Println(shhKeyError)
			return
		}

		// Generate a session token so we can correlate the 2 http calls for generation and retrieval of the certificate
		sessionToken := randomString(randomStringLength)

		fmt.Printf("Requesting certificate to %s/%s\n", org, repo)
		err := requestCertificateCreation(org, repo, sshKey, path.Base(pubKey), sessionToken)

		if err != nil {
			fmt.Println(err)
			return
		}

		// Certificate has been created on the server, let's retrieve it
		for counter := 0; counter < 10; counter++ {
			fmt.Printf("Fetching certificate %d/10\n", counter+1)

			cert, error := fetchCertificate(sessionToken)
			if error == nil {
				// We got the certificate, let's write it to the file
				fmt.Println("Certificate succesfully fetched")

				// id_rsa.pub is the public key file name, the cert file is id_rsa-cert.pub
				outputFile := path.Dir(pubKey) + "/" + strings.TrimSuffix(path.Base(pubKey), ".pub") + "-cert.pub"
				fmt.Printf("Writing certificate to %s\n", outputFile)

				file, err := os.Create(outputFile)
				if err != nil {
					fmt.Printf("Got error while opening the output file. Err: %s", err.Error())
					return
				}
				writer := bufio.NewWriter(file)
				_, err = writer.WriteString(cert)
				if err != nil {
					fmt.Printf("Got error while writing to a file. Err: %s", err.Error())
					return
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
	getCmd.Flags().StringVarP(&pubKey, "pubKey", "k", "", "Public key file")
	getCmd.MarkFlagRequired("pubKey")
	getCmd.Flags().StringVarP(&serverRootURL, "url", "u", "", "The SSH Certificate app's root URL")
	getCmd.MarkFlagRequired("serverRootURL")
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

/*
 * Request the creation of a certificate by sending a repository dispatch event to a repo.
 * The GitHub App listen to this event and will create a certificate.
 * This ensure that the current user is authenticated through GitHub and has write access to the repo.
 */
func requestCertificateCreation(org string, repo string, sshKey SSHKey, pubKeyFileName string, sessionToken string) error {
	client, clientError := gh.RESTClient(nil)
	if clientError != nil {
		return clientError
	}

	response := struct{ Message string }{}
	body := fmt.Sprintf(`{"event_type": "certificate-request", "client_payload": {"sessionToken": "%s", "key": "%s", "title": "%s", "pubKeyFileName": "%s"}}`,
		sessionToken, sshKey.Key, sshKey.Title, pubKeyFileName)
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
 * Retrieve locally the user's public key and make sure it exists on their GitHub profile
 */
func getSSHKey(sshKeyFile string) (SSHKey, error) {
	fmt.Printf("Loading key from file %s\n", sshKeyFile)

	// Read the SSH key from the file
	content, readSSHKeyFileErr := os.ReadFile(sshKeyFile)
	if readSSHKeyFileErr != nil {
		return SSHKey{}, readSSHKeyFileErr
	}

	sshKey := string(content)

	keys, error := getSSHKeys()
	if error != nil {
		return SSHKey{}, error
	}

	if len(keys) == 0 {
		return SSHKey{}, fmt.Errorf("no SSH keys found on your GitHub profile. Please add one")
	}

	// Checking if the public key exists on the user's GitHub profile
	for _, key := range keys {
		if strings.HasPrefix(sshKey, key.Key) {
			fmt.Printf("Found match with key %s on your GitHub profile\n", key.Title)
			return key, nil
		}
	}

	return SSHKey{}, fmt.Errorf("no matching SSH key was found on your GitHub profile. Please add one")
}

/*
 * Retrieve the certificate from the server
 */
func fetchCertificate(sessionToken string) (string, error) {
	client, clientError := gh.RESTClient(nil)
	if clientError != nil {
		return "", clientError
	}

	response := struct {
		Message     string
		Certificate string
	}{}
	body := fmt.Sprintf(`{"sessionToken": "%s"}`, sessionToken)
	bodyReader := bytes.NewReader([]byte(body))
	url := fmt.Sprintf("%s/fetch", serverRootURL)
	postError := client.Post(url, bodyReader, &response)

	if postError != nil {
		return "", postError
	}

	return response.Certificate, nil
}
