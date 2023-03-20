/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"time"

	"github.com/docker/docker/api/types"
	gltr "github.com/gltr-sh/gltr/pkg"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// listTasksCmd represents the listTasks command
var listTasksCmd = &cobra.Command{
	Use:   "list-tasks",
	Short: "List gltr tasks running on an execution platform",
	Run:   listTasks,
}

func init() {
	rootCmd.AddCommand(listTasksCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listTasksCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	listTasksCmd.Flags().StringP("file", "f", "gltr.yaml", "gltr yaml file")
}

// currently, this only supports tasks running on docker
func printTasks(d gltr.DockerExecutionPlatform, c []types.Container) {
	tableData := pterm.TableData{
		[]string{"Container ID", "Container Name", "Start Time"},
	}

	for _, t := range c {
		startTime := time.Unix(t.Created, 0)
		taskID := d.GetTag(t, "gltr-task-id")
		if taskID == nil {
			// this should not really happen but in case it does, we just don't print anything
			continue
		}
		tableData = append(tableData, []string{*taskID, t.Names[0], startTime.Format(time.RFC3339)})
	}

	// Create a fork of the default table, fill it with data and print it.
	// Data can also be generated and inserted later.
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

func listTasks(cmd *cobra.Command, args []string) {

	gltrFilename, _ := cmd.Flags().GetString("file")

	gt, err := readGltrFile(gltrFilename)
	if err != nil {
		pterm.Error.Printf("Error reading gltr file - exiting: %v", err.Error())
		os.Exit(1)
	}

	switch gt.DefaultExecutionPlatform {
	case gltr.Ec2:
		gltr.ListTasksEc2(gt.ProjectName)
	case gltr.EcsFargate:
		ecsConfig := gt.GetExecutionPlatformProjectConfig(gltr.EcsFargate).(gltr.EcsProjectConfig)
		gltr.ListTasksEcs(ecsConfig.ClusterName)
	case gltr.Docker:
		pterm.Info.Printf("Obtaining running task information from local docker engine\n")
		dockerExecutionPlatform := gltr.DockerExecutionPlatform{}
		tasks, err := dockerExecutionPlatform.ListTasks()
		if err != nil {
			pterm.Error.Printf("Error obtaining task list %v\n", err)
			os.Exit(1)
		}
		if len(tasks) == 0 {
			pterm.Info.Printf("No running tasks found\n")
			return
		}
		pterm.Success.Printf("Obtained task list from local docker engine\n")
		pterm.Println()
		printTasks(dockerExecutionPlatform, tasks)
	default:
		pterm.Error.Printf("Unknown execution platform %v\n", gt.DefaultExecutionPlatform)
		os.Exit(1)
	}
}
