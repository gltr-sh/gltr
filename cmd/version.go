/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print gltr version",
	Long:  "Print gltr version",
	Run:   version,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func version(cmd *cobra.Command, args []string) {
	info, _ := debug.ReadBuildInfo()

	// get vcs.time
	var buildTime, version string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.time":
			buildTime = s.Value
		case "vcs.revision":
			version = s.Value
		default:
			continue
		}
	}

	// want to get the following - go, vcs.version, vcs.time
	fmt.Printf("gltr version %v (built %v with %v)\n", version, buildTime, info.GoVersion)
}
