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

// defaultExecutionEnvironmentCmd represents the defaultExecutionEnvironment command
var projectDefaultExecutionPlatformCmd = &cobra.Command{
	Use:   "default-execution-platform",
	Short: "Select default execution environment for project",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: defaultExecutionPlatform,
}

func init() {
	projectCmd.AddCommand(projectDefaultExecutionPlatformCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// defaultExecutionEnvironmentCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// defaultExecutionEnvironmentCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	projectDefaultExecutionPlatformCmd.Flags().StringP("file", "f", "gltr.yaml", "Gltr yaml file")
}

func defaultExecutionPlatform(cmd *cobra.Command, args []string) {

	gltrFilename, _ := cmd.Flags().GetString("file")

	gt, err := readGltrFile(gltrFilename)
	if err != nil {
		log.Printf("Error reading gltr file - exiting: %v", err.Error())
		os.Exit(1)
	}

	gt.DefaultExecutionPlatform = chooseDefaultExecutionPlatform(gt)

	writeGltrFile(gltrFilename, gt)

	fmt.Printf("\n")
	fmt.Printf("Updated gltr file written for project\n")
}
