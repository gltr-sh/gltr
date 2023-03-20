/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"
	"os"

	gltr "github.com/gltr-sh/gltr/pkg"
	"github.com/spf13/cobra"
)

// showTaskCmd represents the showTask command
var showTaskCmd = &cobra.Command{
	Use:   "show-task",
	Short: "Show detailed informatoin relating to a specific gltr task",
	Run:   showTask,
}

func init() {
	rootCmd.AddCommand(showTaskCmd)

	showTaskCmd.Flags().String("task-id", "", "ID of task to be shown")
	showTaskCmd.Flags().StringP("file", "f", "gltr.yaml", "gltr yaml file")
}

func showTask(cmd *cobra.Command, args []string) {
	taskID, _ := cmd.Flags().GetString("task-id")
	if taskID == "" {
		log.Printf("Error: no task-id specified")
		os.Exit(1)
	}

	gltrFilename, _ := cmd.Flags().GetString("file")

	gt, err := readGltrFile(gltrFilename)
	if err != nil {
		log.Printf("Error reading gltr file - exiting: %v", err.Error())
		os.Exit(1)
	}

	switch gt.DefaultExecutionPlatform {
	case gltr.Ec2:
		gltr.ShowTaskEc2(taskID)
	case gltr.EcsFargate:
		ecsConfig := gt.GetExecutionPlatformProjectConfig(gltr.EcsFargate).(gltr.EcsProjectConfig)
		gltr.ShowTaskEcs(ecsConfig.ClusterName, taskID)
	}
}
