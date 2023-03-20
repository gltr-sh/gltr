/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	gltr "github.com/gltr-sh/gltr/pkg"
	"github.com/spf13/cobra"
)

// addUserCmd represents the addUser command
var addUserCmd = &cobra.Command{
	Use:   "add-user",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
}

func init() {
	projectCmd.AddCommand(addUserCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addUserCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addUserCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	addUserCmd.Flags().StringP("file", "f", "gltr.yaml", "gltr yaml file")
	addUserCmd.Run = projectAddUser
}

func getUserData() gltr.User {
	user := gltr.ReadTextInput("Enter user name", "", "")
	email := gltr.ReadTextInput("Enter email address", "", "")
	// lookupSshKeysWithEmailAddr(email)
	// githubUsername := readTextInput("Enter GitHub username", "", "")
	// keys := lookupSSHKeysWithUsername(githubUsername)

	sshKey := gltr.ReadTextInput("Enter SSH Key ", "", "")

	u := gltr.User{
		Name:   user,
		Email:  email,
		SshKey: sshKey,
	}

	return u
}

func projectAddUser(cmd *cobra.Command, args []string) {
	u := getUserData()
	fmt.Printf("user = %v\n", u)

	gltrFilename, _ := addUserCmd.Flags().GetString("file")
	fmt.Printf("filename = %v\n", gltrFilename)
	gt, err := readGltrFile(gltrFilename)
	if err != nil {
		fmt.Printf("error reading gltr file %v", err)
		os.Exit(1)
	}
	fmt.Printf("users = %v\n", gt.Users)
	gt.Users = append(gt.Users, u)
	fmt.Printf("users = %v\n", gt.Users)
	_ = writeGltrFile(gltrFilename, gt)
}
