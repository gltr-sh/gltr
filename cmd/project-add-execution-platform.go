/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/package cmd

import (
	"fmt"
	"log"
	"os"

	gltr "github.com/gltr-sh/gltr/pkg"
	"github.com/spf13/cobra"
)

// addExecutionPlatformCmd represents the addExecutionPlatform command
var projectAddExecutionPlatformCmd = &cobra.Command{
	Use:   "add-execution-platform",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: projectAddExecutionPlatform,
}

func init() {
	projectCmd.AddCommand(projectAddExecutionPlatformCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addExecutionPlatformCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addExecutionPlatformCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	projectAddExecutionPlatformCmd.Flags().StringP("file", "f", "gltr.yaml", "gltr yaml file")
}

func projectAddExecutionPlatform(cmd *cobra.Command, args []string) {
	// read gltr config
	gltrConfigDir := getGltrConfigDir()
	gltrConfig, err := readGltrConfig(gltrConfigDir)
	if err != nil {
		fmt.Printf("Error reading gltr config: %s\n", err)
		os.Exit(1)
	}

	var availableExecutionPlatforms []string
	for _, e := range gltrConfig.ExecutionPlatforms {
		platformName := e.Type.ToString()
		availableExecutionPlatforms = append(availableExecutionPlatforms, platformName)
	}

	gltrFilename, _ := cmd.Flags().GetString("file")

	gt, err := readGltrFile(gltrFilename)
	if err != nil {
		log.Printf("Error reading gltr file - exiting: %v", err.Error())
		os.Exit(1)
	}

	// need to add logic to determine the available but unconfigured platforms...
	// and then simply choose one if there is only one option...
	platformToAdd := gltr.ReadOptionInput(
		"Choose From Available Execution Platforms:",
		"",
		availableExecutionPlatforms,
	)

	// convert string to type
	platformTypeToAdd, _ := gltr.ParseExecutionPlatformType(platformToAdd)
	var newPlatformConfig gltr.ExecutionPlatformProjectConfig
	switch platformTypeToAdd {
	case gltr.Docker:
		newPlatformConfig, err = gltr.ProjectAddDockerExecutionPlatform(gltrConfig)
		if err != nil {
			fmt.Printf("Error adding docker execution platform: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Adding docker\n")
	case gltr.EcsFargate:
		newPlatformConfig, err = gltr.ProjectAddEcsFargateExecutionPlatform(gt, gltrConfig)
		if err != nil {
			fmt.Printf("Error adding ecs fargate execution platform: %s\n", err)
			os.Exit(1)
		}
		err = writeGltrConfig(gltrConfigDir, gltrConfig)
		if err != nil {
			fmt.Printf("Error writing gltr config: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("New gltr configuration written to file\n")
	case gltr.Ec2:
		newPlatformConfig, err = gltr.ProjectAddEc2ExecutionPlatform(gt, gltrConfig)
		if err != nil {
			fmt.Printf("Error adding ecs fargate execution platform: %s\n", err)
			os.Exit(1)
		}
		err = writeGltrConfig(gltrConfigDir, gltrConfig)
		if err != nil {
			fmt.Printf("Error writing gltr config: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("New gltr configuration written to file\n")
	case gltr.GcpComputeEngine:
		fmt.Printf("GcpComputeEngine support not yet implemented\n")
		break
	default:
		// this should never happen
		fmt.Printf("Error adding execution platform: unrecognized execution platform type\n")
		os.Exit(1)
	}

	// new set of available execution platforms
	// write new execution platform to
	gt.ExecutionPlatformConfigs = append(gt.ExecutionPlatformConfigs, newPlatformConfig)

	gt.DefaultExecutionPlatform = chooseDefaultExecutionPlatform(gt)

	writeGltrFile(gltrFilename, gt)

	fmt.Printf("\n")
	fmt.Printf("Updated gltr file written for project\n")
}

func chooseDefaultExecutionPlatform(gt gltr.Task) gltr.ExecutionPlatformType {
	var configuredExecutionPlatforms []string
	for _, e := range gt.ExecutionPlatformConfigs {
		platformName := e.Type.ToString()
		configuredExecutionPlatforms = append(configuredExecutionPlatforms, platformName)
	}

	defaultExecutionPlatform := gltr.ReadOptionInput(
		"Default Execution Platform:",
		gt.DefaultExecutionPlatform.ToString(),
		configuredExecutionPlatforms,
	)
	defaultExecutionPlatformParsed, _ := gltr.ParseExecutionPlatformType(defaultExecutionPlatform)
	return defaultExecutionPlatformParsed

}
