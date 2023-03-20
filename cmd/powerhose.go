/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// powerhoseCmd represents the powerhose command
var powerhoseCmd = &cobra.Command{
	Use:   "powerhose",
	Short: "A powerful cleanup command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: gltrPowerhose,
}

func init() {
	rootCmd.AddCommand(powerhoseCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// powerhoseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// powerhoseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	powerhoseCmd.Flags().Bool("aws", false, "Powerhose AWS")
	powerhoseCmd.Flags().Bool("azure", false, "Powerhose Azure")
}

func gltrPowerhose(cmd *cobra.Command, args []string) {
	// powerhoseAws, _ := cmd.Flags().GetBool("aws")
	powerhoseAzure, _ := cmd.Flags().GetBool("azure")

	gltrConfigDir := getGltrConfigDir()
	config, err := readGltrConfig(gltrConfigDir)
	if err != nil {
		fmt.Printf("No existing configuration - unable to powerhose configuration...%v\n", err)
		os.Exit(1)
	}

	if powerhoseAzure {
		log.Printf("Azure not yet supported - unable to powerhose.")
	}
	// if powerhoseAws {
	// 	fmt.Printf("This will do the following:\n")
	// 	fmt.Printf("- remove gltr ECS cluster\n")
	// 	fmt.Printf("- remove gltr VPC, internet gateway and subnet\n")
	// 	err = gltr.PowerhoseAws(config.Aws)
	// 	if err != nil {
	// 		log.Printf("Error powerhosing AWS: %v\n", err.Error())
	// 		os.Exit(1)
	// 	}
	// }

	// awsConfig := config.Aws
	// // remove the settings
	// awsConfig.ClusterName = ""
	// awsConfig.VpcID = ""
	// awsConfig.SubnetID = ""
	// awsConfig.IgwID = ""

	err = writeGltrConfig(gltrConfigDir, config)
	if err != nil {
		fmt.Printf("Error writing configuration info: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Updated configuration written to %v\n", gltrConfigDir)
}
