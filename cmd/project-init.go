/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	gltr "github.com/gltr-sh/gltr/pkg"
	git "github.com/go-git/go-git/v5"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/tcnksm/go-gitconfig"
)

// projectCmd represents the project command
var projectInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gltr project",
	Long:  "Initialize gltr project",
}

func init() {
	projectInitCmd.Run = projectInit
	projectCmd.AddCommand(projectInitCmd)
	projectInitCmd.Flags().StringP("file", "f", "gltr.yaml", "gltr yaml file")
}

func checkIfDirIsGitRepo(dir string) (string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		fmt.Printf("Error determining if current directory is a git repo: %v", err)
		return "", err
	}
	remotes, err := repo.Remotes()
	if err != nil {
		fmt.Printf("Error obtaining remotes from current repo: %v", err)
		return "", err
	}
	// assume that there is at least one remote - need to check if err=nil means
	// there are no remotes...
	firstRemote := remotes[0]
	remoteUrls := firstRemote.Config().URLs
	firstRemoteURL := remoteUrls[0]
	return firstRemoteURL, nil
}

// this function lists the public keys in ~/.ssh; the user can choose one
// of these or else other; if the user chooses other, then she must enter
// a key diectly...
func getSSHKey() (sshKey string) {
	// list public keys in ~/.ssh
	homeDir := os.Getenv("HOME")
	globPath := path.Join(homeDir, ".ssh", "*.pub")
	files, err := filepath.Glob(globPath)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	OtherString := "Other..."
	files = append(files, OtherString)
	// options
	response := gltr.ReadOptionInput("Choose SSH Public Key", "", files)
	if response == OtherString {
		sshKey = gltr.ReadTextInput("Enter SSH Public Key ", "", "")
		return
	}

	// read file into key
	bytes, err := ioutil.ReadFile(response)
	sshKey = string(bytes)
	return
}

// this is for the case in which there is no gltr config - it tries to create
// the minimal gltr config which focuses on supporting local execution. If no
// docker engine exists, then it fails
// minim
func initMinimalGltrConfig(configDir string) error {
	dockerConfig, err := gltr.ConfigAddDockerExecutionPlatform(gltr.Config{})
	if err != nil {
		fmt.Printf("Error adding docker execution platform - unable to initialize gltr: %v", err)
		os.Exit(1)
	}

	executionPlatforms := []gltr.ExecutionPlatform{
		dockerConfig,
	}

	// get user details from git config
	gitUser, err := gitconfig.Global("user.name")
	gitEmail, err := gitconfig.Global("user.email")

	user := gltr.ReadTextInput("Enter user name", gitUser, gitUser)
	email := gltr.ReadTextInput("Enter email address", gitEmail, gitEmail)

	sshKey := getSSHKey()
	u := gltr.User{
		Name:   user,
		Email:  email,
		SshKey: sshKey,
	}

	gltrConfig := gltr.Config{
		ExecutionPlatforms: executionPlatforms,
		User:               u,
		LastUpdate:         time.Now(),
	}

	err = writeGltrConfig(configDir, gltrConfig)
	if err != nil {
		fmt.Printf("Error writing configuration to file: %v\n", err)
		return err
	}
	fmt.Printf("Configuration written to %v\n", configDir)

	return nil
}

func initializeProjectWithConfig(config gltr.Config, gltrConfigDir, repoName, gltrFilename string) error {

	projectID := uuid.New().String()
	fmt.Printf("Creating new gltr project with ID: %v\n", projectID)

	publicKey, privateKey, err := generateKeyPair()
	if err != nil {
		fmt.Printf("Error generating key pair: %v", err)
		return err
	}
	keyDirectory := path.Join(gltrConfigDir, "secrets", projectID)

	writeKeysToDirectory(keyDirectory, projectID, publicKey, privateKey)

	shortProjectName := extractProjectShortNameFromRepo(repoName)
	// set up some defaults
	project := gltr.Task{
		ProjectID:      projectID,
		ProjectName:    shortProjectName,
		ContainerImage: defaultContainerImage,
		GpuRequired:    false,
		Users:          []gltr.User{config.User},
		GitRepo:        repoName,
		ExecutionPlatformConfigs: []gltr.ExecutionPlatformProjectConfig{
			{
				Type: gltr.Docker,
				Configuration: gltr.DockerProjectConfig{
					GpuEnabled: false,
				},
			},
		},
		ProjectPublicKey: string(*publicKey),
	}

	updatedProject := initializeProjectWithDefaults(project, config)

	err = writeGltrFile(gltrFilename, updatedProject)
	if err != nil {
		fmt.Printf("Error writing gltr file: %v\n", err)
		return err
	}
	return nil
}

func projectInit(cmd *cobra.Command, args []string) {

	// first check if there is a gltr configuration file in thie directory....
	gltrFilename, _ := projectInitCmd.Flags().GetString("file")

	fileExists := fileExists(gltrFilename)
	if fileExists {
		fmt.Printf(
			"gltr project config file (%v) already exists in this directory - will not overwrite - exiting...\n",
			gltrFilename,
		)
		os.Exit(1)
	}

	// first check if the current dir is a git repo; if not, we quit
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error obtaining current directory: %v", err)
		os.Exit(1)
	}

	repoName, err := checkIfDirIsGitRepo(currentDir)
	if err != nil {
		fmt.Printf("Current directory is not a git repo - exiting...: %v", err)
		os.Exit(1)
	}
	fmt.Printf("Current directory is valid git repo (%v)\n", repoName)

	// check if gltr has already been initialized...
	gltrConfigDir := getGltrConfigDir()
	_, err = readGltrConfig(gltrConfigDir)
	if err != nil {
		fmt.Printf("gltr system configuration does not exist -  initializing...\n\n")
		initMinimalGltrConfig(gltrConfigDir)
	} else {
		fmt.Printf("gltr system configuration found in %v\n", gltrConfigDir)
	}

	// if we get here, we should have a valid gltr config which has either been
	// just generated or existed a priori
	config, err := readGltrConfig(gltrConfigDir)
	if err != nil {
		fmt.Printf("Error reading gltr config - exiting...: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nInitializing gltr project...\n")
	// now we enter the project initialization proper...
	initializeProjectWithConfig(config, gltrConfigDir, repoName, gltrFilename)

}
