/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// showCmd represents the show command
var showCmd = &cobra.Command{
	Use:   "show",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: configShow,
}

func init() {
	configCmd.AddCommand(showCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// showCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// showCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func configShow(cmd *cobra.Command, args []string) {
	gltrConfigDir := getGltrConfigDir()
	gltrConfig, err := readGltrConfig(gltrConfigDir)
	if err != nil {
		fmt.Printf("Error reading gltr config: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("---\n")
	marshalledConfig, err := yaml.Marshal(gltrConfig)
	fmt.Printf("%v", string(marshalledConfig))

	// fmt.Println("config show called")
}
