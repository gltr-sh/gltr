/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	gltr "github.com/gltr-sh/gltr/pkg"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gltr",
	Long:  "Initializes gltr with user input; sets up basic user data and checks availability of docker",
	Run:   gltrInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	// FIXME
	initCmd.Flags().Bool("aws", false, "Initialize AWS")
	initCmd.Flags().Bool("azure", false, "Initialize Azure")
}

// this function checks if there is a valid AWS configuration
// for now, it just checks if a cluster name, vpc, subnet id
// have been created
func checkAWSConfig(c gltr.AWSConfig) bool {
	// if c.ClusterName == "" {
	// 	return false
	// }
	if c.VpcID == "" {
		return false
	}
	if c.SubnetID == "" {
		return false
	}
	return true
}

func gltrInit(cmd *cobra.Command, args []string) {
	// initializeAws, _ := cmd.Flags().GetBool("aws")
	initializeAzure, _ := cmd.Flags().GetBool("azure")

	gltrConfigDir := getGltrConfigDir()
	config, err := readGltrConfig(gltrConfigDir)
	if err != nil {
		fmt.Printf("No existing configuration - creating new configuration...\n")
	}

	if initializeAzure {
		fmt.Printf("Azure not yet supported - unable to initialize.")
	}

	writeGltrConfig(gltrConfigDir, config)
	fmt.Printf("Configuration written to %v\n", gltrConfigDir)
}
