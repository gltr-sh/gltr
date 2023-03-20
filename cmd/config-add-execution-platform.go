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

// addExecutionPlatformCmd represents the addExecutionPlatform command
var configAddExecutionPlatformCmd = &cobra.Command{
	Use:   "add-execution-platform",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: configAddExecutionPlatform,
}

func init() {
	configCmd.AddCommand(configAddExecutionPlatformCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addExecutionPlatformCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addExecutionPlatformCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func configAddExecutionPlatform(cmd *cobra.Command, args []string) {
	// read gltr config
	gltrConfigDir := getGltrConfigDir()
	gltrConfig, err := readGltrConfig(gltrConfigDir)
	if err != nil {
		fmt.Printf("Error reading gltr config: %s\n", err)
		os.Exit(1)
	}

	var configuredExecutionPlatforms []string
	for _, e := range gltrConfig.ExecutionPlatforms {
		platformName := e.Type.ToString()
		configuredExecutionPlatforms = append(configuredExecutionPlatforms, platformName)
	}

	fmt.Printf("Execution platforms configured:\n")
	for _, c := range configuredExecutionPlatforms {
		fmt.Printf("- %v\n", c)
	}
	fmt.Printf("\n")

	var unconfiguredExecutionPlatforms []string
	for platformType, p := range gltr.ExecutionPlatformMap {
		if platformType != gltr.UnknownPlatform {
			// check if p is in the configuredExecutionPlatforms array
			found := false
			for _, c := range configuredExecutionPlatforms {
				if c == p {
					found = true
					break
				}
			}
			if found == false {
				unconfiguredExecutionPlatforms = append(unconfiguredExecutionPlatforms, p)
			}
		}
	}

	if len(unconfiguredExecutionPlatforms) == 0 {
		fmt.Printf("All available execution platforms configured - nothing to do.")
		os.Exit(0)
	}

	platformToAdd := gltr.ReadOptionInput("Choose Execution Platform:", "", unconfiguredExecutionPlatforms)

	// convert string to type
	platformTypeToAdd, _ := gltr.ParseExecutionPlatformType(platformToAdd)
	switch platformTypeToAdd {
	case gltr.Docker:
		// this looks buggy - FIXME
		dockerConfig, err := gltr.ConfigAddDockerExecutionPlatform(gltrConfig)
		if err != nil {
			fmt.Printf("Error adding docker execution platform: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Adding docker\n")
		fmt.Printf("config: %v\n", dockerConfig)
	case gltr.Ec2:
		gltrConfig, err := gltr.ConfigAddEc2ExecutionPlatform(gltrConfig)
		if err != nil {
			fmt.Printf("Error adding ec2 execution platform: %s\n", err)
			os.Exit(1)
		}
		err = writeGltrConfig(gltrConfigDir, gltrConfig)
		if err != nil {
			fmt.Printf("Error writing gltr config: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("New gltr configuration written to file\n")
	case gltr.EcsFargate:
		gltrConfig, err := gltr.ConfigAddEcsFargateExecutionPlatform(gltrConfig)
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
		fmt.Printf("Adding GcpComputeEngine\n")
	default:
		// this should never happen
		fmt.Printf("Error adding execution platform: unrecognized execution platform type\n")
		os.Exit(1)
	}
}
