/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"
	"os"

	gltr "github.com/gltr-sh/gltr/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// killTaskCmd represents the killTask command
var killTaskCmd = &cobra.Command{
	Use:   "kill-task",
	Short: "Terminate a specified task",
	Run:   killTask,
}

func init() {
	rootCmd.AddCommand(killTaskCmd)

	killTaskCmd.Flags().String("task-id", "", "ID of task to be terminated")
	killTaskCmd.Flags().StringP("file", "f", "gltr.yaml", "gltr yaml file")
}

func killTask(cmd *cobra.Command, args []string) {
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
		gltr.KillTaskEc2(taskID)
	case gltr.EcsFargate:
		ecsConfig := gt.GetExecutionPlatformProjectConfig(gltr.EcsFargate).(gltr.EcsProjectConfig)
		gltr.KillTaskEcs(ecsConfig.ClusterName, taskID)
	case gltr.Docker:
		pterm.Info.Printf("Terminating task %v on local docker engine\n", taskID)
		dockerExecutionPlatform := gltr.DockerExecutionPlatform{}
		err := dockerExecutionPlatform.KillTask(taskID)
		if err != nil {
			pterm.Error.Printf("Error terminating task %v\n", err)
			os.Exit(1)
		}
		pterm.Success.Printf("Task %v terminated\n", taskID)
	default:
		pterm.Error.Printf("Unknown execution platform %v\n", gt.DefaultExecutionPlatform)
		os.Exit(1)
	}
}
