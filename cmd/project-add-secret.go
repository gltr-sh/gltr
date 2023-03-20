/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	sshage "github.com/Mic92/ssh-to-age"
	"go.mozilla.org/sops/v3"
	sopsaes "go.mozilla.org/sops/v3/aes"
	"go.mozilla.org/sops/v3/age"
	"go.mozilla.org/sops/v3/cmd/sops/common"
	sopsyaml "go.mozilla.org/sops/v3/stores/yaml"
	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"
)

// addSecretCmd represents the addSecret command
var projectAddSecretCmd = &cobra.Command{
	Use:   "add-secret",
	Short: "Add a secret to a project",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: projectAddSecret,
}

func init() {
	projectCmd.AddCommand(projectAddSecretCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addSecretCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addSecretCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	projectAddSecretCmd.Flags().StringP("file", "f", "gltr.yaml", "gltr yaml file")
}

// reads in the private key and returns it
func getPublicKey(configDir, projectID string) ([]byte, error) {

	// the private key is stored in gltrConfigDir/secrets/projectid/projectid
	publicKeyShortFilename := fmt.Sprintf("%v.pub", projectID)
	publicKeyFileName := path.Join(configDir, "secrets", projectID, publicKeyShortFilename)

	file, err := os.Open(publicKeyFileName)
	if err != nil {
		return nil, fmt.Errorf("error reading private key file: %w", err)
	}
	sshKey, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading private key file: %w", err)
	}
	return sshKey, nil
}

// reads in the private key and returns it
func getPrivateKey(configDir, projectID string) ([]byte, error) {

	// the private key is stored in gltrConfigDir/secrets/projectid/projectid
	privateKeyFileName := path.Join(configDir, "secrets", projectID, projectID)

	file, err := os.Open(privateKeyFileName)
	if err != nil {
		return nil, fmt.Errorf("error reading private key file: %w", err)
	}
	sshKey, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading private key file: %w", err)
	}
	return sshKey, nil
}

func initializeSecretFileWithSecret(gltrSecretFileName, secretFile string) {
	fmt.Printf("Encrypting file: %s\n", secretFile)

	// read glattefile
	gltrFilename, _ := addUserCmd.Flags().GetString("file")
	// fmt.Printf("filename = %v\n", gltrFilename)
	gt, err := readGltrFile(gltrFilename)
	if err != nil {
		fmt.Printf("error reading gltr file %v", err)
		os.Exit(1)
	}

	// read private key for this project...
	gltrConfigDir := getGltrConfigDir()
	sshPublicKey, err := getPublicKey(gltrConfigDir, gt.ProjectID)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}

	// create age key from gltr key...this should not be written to anywhere
	// on the filesystem
	ageRecipientPublicKey, err := sshage.SSHPublicKeyToAge(sshPublicKey)
	fmt.Printf("Encrypting with project pubkey: %v\n", *ageRecipientPublicKey)

	masterKey, err := age.MasterKeyFromRecipient(*ageRecipientPublicKey)
	if err != nil {
		fmt.Printf("Error converting age key...\n")
		os.Exit(1)
	}

	keygroup := sops.KeyGroup{
		masterKey,
	}

	// so next we need to create the file...
	// so this is not following the right logic..

	secretFileContents, err := ioutil.ReadFile(secretFile)

	b64EncodedFile := base64.StdEncoding.EncodeToString(secretFileContents)

	branch := sops.TreeBranch{
		sops.TreeItem{
			Key:   secretFile,
			Value: string(b64EncodedFile),
		},
	}

	tree := sops.Tree{
		Branches: []sops.TreeBranch{branch},
		Metadata: sops.Metadata{
			KeyGroups:         []sops.KeyGroup{keygroup},
			UnencryptedSuffix: "",
			EncryptedSuffix:   "",
			UnencryptedRegex:  "",
			EncryptedRegex:    "",
			Version:           "version-string",
		},
		FilePath: "test-path",
	}

	dataKey, errs := tree.GenerateDataKey()
	if len(errs) > 0 {
		err = fmt.Errorf("Could not generate data key: %s", errs)
		os.Exit(1)
	}

	err = common.EncryptTree(common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    &tree,
		Cipher:  sopsaes.NewCipher(),
	})
	if err != nil {
		fmt.Printf("err = %v\n", err)
		os.Exit(1)
	}

	outputStore := sopsyaml.Store{}
	encryptedFile, err := outputStore.EmitEncryptedFile(tree)
	if err != nil {
		fmt.Printf("Error encrypting file: %v...\n", err)
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Error encrypting file: %v...\n", err)
		os.Exit(1)
	}
	err = os.WriteFile(gltrSecretFileName, encryptedFile, 0600)
	if err != nil {
		fmt.Printf("Error writing encrypted file: %v...\n", err)
		os.Exit(1)
	}
	fmt.Printf("Secret file created in %v and initialized with secret.\n", gltrSecretFileName)
}

// this should be called with the name of a file as an argument
func addSecretExistingFile(gltrSecretFileName, secretFileName string) error {

	// read glattefile
	gltrFilename, _ := addUserCmd.Flags().GetString("file")
	// fmt.Printf("filename = %v\n", gltrFilename)
	gt, err := readGltrFile(gltrFilename)
	if err != nil {
		fmt.Printf("error reading gltr file %v", err)
		os.Exit(1)
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
	_, ok := decryptedMap[secretFileName].(string)
	if ok {
		fmt.Printf("Warning: key already defined in secret file - overwriting...\n")
	}

	newSecretFileContents, err := ioutil.ReadFile(secretFileName)
	b64EncodedFile := base64.StdEncoding.EncodeToString(newSecretFileContents)

	decryptedMap[secretFileName] = b64EncodedFile

	newBranch := sops.TreeBranch{}
	for k, v := range decryptedMap {
		newBranch = append(newBranch, sops.TreeItem{Key: k, Value: v})
	}

	newTree := sops.Tree{
		Branches: []sops.TreeBranch{newBranch},
		Metadata: tree.Metadata,
		FilePath: "test-path",
	}

	dataKey, errs := newTree.GenerateDataKey()
	if len(errs) > 0 {
		err = fmt.Errorf("Could not generate data key: %s", errs)
		os.Exit(1)
	}

	err = common.EncryptTree(common.EncryptTreeOpts{
		DataKey: dataKey,
		Tree:    &newTree,
		Cipher:  sopsaes.NewCipher(),
	})
	if err != nil {
		fmt.Printf("err = %v\n", err)
		os.Exit(1)
	}

	outputStore := sopsyaml.Store{}
	encryptedFile, err := outputStore.EmitEncryptedFile(newTree)
	if err != nil {
		fmt.Printf("Error encrypting file: %v...\n", err)
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Error encrypting file: %v...\n", err)
		os.Exit(1)
	}
	err = os.WriteFile(gltrSecretFileName, encryptedFile, 0600)
	if err != nil {
		fmt.Printf("Error writing encrypted file: %v...\n", err)
		os.Exit(1)
	}
	fmt.Printf("Updated gltr secrets file with new secret.\n")

	os.Remove(agePrivateKeyFile)
	return nil
}

// this should be called with the name of a file as an argument
func projectAddSecret(cmd *cobra.Command, args []string) {
	switch {
	case len(args) == 0:
		fmt.Printf("No file to encrypt...exiting\n")
		os.Exit(1)
	case len(args) > 1:
		fmt.Printf("Arguments > 1...exiting\n")
		os.Exit(1)
	}

	// first we check if the secret file exists; if not, there is nothing we
	// need to do...
	secretFile := args[0]
	_, err := os.Stat(secretFile)
	if err != nil {
		// this occurs if the file is not found...
		fmt.Printf("Unable to find file %s...exiting\n", secretFile)
		os.Exit(1)
	}

	gltrSecretFileName := "gltr-secrets.yaml"
	_, err = os.Stat(gltrSecretFileName)
	if err != nil {
		// this occurs if the file is not found...
		initializeSecretFileWithSecret(gltrSecretFileName, secretFile)
		return
	}

	addSecretExistingFile(gltrSecretFileName, secretFile)

	// fmt.Printf("Encrypted file = %v\n", encryptedFile)
}
