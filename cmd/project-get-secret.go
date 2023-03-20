/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	sshage "github.com/Mic92/ssh-to-age"
	"github.com/spf13/cobra"
	sopsaes "go.mozilla.org/sops/v3/aes"
	sopsyaml "go.mozilla.org/sops/v3/stores/yaml"
	"gopkg.in/yaml.v3"
)

// getSecretCmd represents the getSecret command
var projectGetSecretCmd = &cobra.Command{
	Use:   "get-secret",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
}

func init() {
	projectGetSecretCmd.Run = projectGetSecret
	projectCmd.AddCommand(projectGetSecretCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getSecretCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getSecretCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	projectGetSecretCmd.Flags().StringP("file", "f", "gltr.yaml", "Gltr yaml file")
	projectGetSecretCmd.Flags().StringP("output", "o", "", "Output file")
}

// this should be called with the name of a file as an argument
func projectGetSecret(cmd *cobra.Command, args []string) {
	switch {
	case len(args) == 0:
		fmt.Printf("No secret to get...exiting\n")
		os.Exit(1)
	case len(args) > 1:
		fmt.Printf("Arguments > 1...exiting\n")
		os.Exit(1)
	}

	// read glattefile
	gltrFilename, _ := projectGetSecretCmd.Flags().GetString("file")
	// fmt.Printf("filename = %v\n", gltrFilename)
	gt, err := readGltrFile(gltrFilename)
	if err != nil {
		fmt.Printf("error reading gltr file %v", err)
		os.Exit(1)
	}

	secretOutputFilename, err := projectGetSecretCmd.Flags().GetString("output")
	if len(secretOutputFilename) == 0 {
		fmt.Printf("No output file specified...writing secret to console...\n")
	}

	// read public key for this project...
	gltrConfigDir := getGltrConfigDir()

	sshPrivateKey, err := getPrivateKey(gltrConfigDir, gt.ProjectID)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	// create age key from gltr key...this should not be written to anywhere
	// on the filesystem
	ageRecipientPrivateKey, _, err := sshage.SSHPrivateKeyToAge(sshPrivateKey, nil)
	fmt.Printf("Age Private Key generated\n")

	// ugh - too much hardcoding...
	agePrivateKeyFile := "/tmp/age-key.txt"
	os.WriteFile(agePrivateKeyFile, []byte(*ageRecipientPrivateKey), 0600)
	gltrSecretFile := "gltr-secrets.yaml"
	os.Setenv("SOPS_AGE_KEY_FILE", agePrivateKeyFile)

	secretFileContents, err := ioutil.ReadFile(gltrSecretFile)
	if err != nil {
		fmt.Printf("Error reading secrets file - exiting...\n")
		os.Exit(1)
	}

	gltrSecretStore := sopsyaml.Store{}
	tree, err := gltrSecretStore.LoadEncryptedFile(secretFileContents)
	if err != nil {
		fmt.Printf("Error loading encrypted file %v - exiting...\n", err)
		os.Exit(1)
	}
	// fmt.Printf("tree: %v\n", tree)

	key, err := tree.Metadata.GetDataKey()
	if err != nil {
		fmt.Printf("Error getting data key%v - exiting...\n", err)
		os.Exit(1)
	}

	// Decrypt the tree
	cipher := sopsaes.NewCipher()
	mac, err := tree.Decrypt(key, cipher)
	if err != nil {
		fmt.Printf("Error decrypting tree %v - exiting...\n", err)
		os.Exit(1)
	}

	// Compute the hash of the cleartext tree and compare it with
	// the one that was stored in the document. If they match,
	// integrity was preserved
	originalMac, err := cipher.Decrypt(
		tree.Metadata.MessageAuthenticationCode,
		key,
		tree.Metadata.LastModified.Format(time.RFC3339),
	)
	if originalMac != mac {
		fmt.Printf("Failed to verify data integrity. expected mac %q, got %q", originalMac, mac)
		os.Exit(1)
	}

	// fmt.Printf("gltrSecretStore: %v\n", gltrSecretStore)

	plainTextFile, err := gltrSecretStore.EmitPlainFile(tree.Branches)
	if err != nil {
		fmt.Printf("Error decrypting file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully decrypted secrets\n")

	decryptedMap := make(map[string]interface{})
	err = yaml.Unmarshal(plainTextFile, &decryptedMap)
	if err != nil {
		fmt.Printf("Error unmarshling file: %v\n", err)
		os.Exit(1)
	}
	value, ok := decryptedMap[args[0]].(string)
	if !ok {
		fmt.Printf("Key not defined in secrets file - nothing to do\n")
	} else {
		if secretOutputFilename != "" {
			byteArray, _ := base64.StdEncoding.DecodeString(value)
			if err := ioutil.WriteFile(secretOutputFilename, []byte(byteArray), 0600); err != nil {
				fmt.Printf("Error writing secret to file: %v\n", err)
			} else {
				fmt.Printf("Secret written to %v\n", secretOutputFilename)
			}
		} else {
			fmt.Printf("%v (base64 encoded) = %v\n", args[0], value)
		}
	}

	// this should be in a defer...
	os.Remove(agePrivateKeyFile)
}
