/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gltr",
	Short: "A lightweight CLI tool for frictionless data science",
	Long: `gltr is a tool which removes friction from data science workflows.

gltr enables data science notebooks to be run locally or on different
execution platforms, offering a consistent development experience for
interacting with the data and the workflow. gltr combines git, container
technologies, jupyter, ssh and vscode to deliver this experience. `,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
