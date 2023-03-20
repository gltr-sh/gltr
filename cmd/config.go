/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure gltr",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Please specify subcommand for config")
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
