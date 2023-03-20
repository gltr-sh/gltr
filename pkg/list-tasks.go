package gltr

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/jedib0t/go-pretty/v6/table"
)

// performs a run on AWS. Assumes the following:
// - AWS credenials are available
// - AWS has been initialized as described elswhere
func ListTasksEcs(clusterName string) {

	_, ecsClient, err := getEcsClient()
	if err != nil {
		fmt.Printf("Error initializing AWS session: %v\n", err)
		os.Exit(1)
	}

	cluster, err := getCluster(ecsClient, clusterName)
	if err != nil {
		fmt.Printf("Error obtaining cluster: %v\n", err)
		os.Exit(1)
	}

	clusterArn := *cluster.ClusterArn
	fmt.Printf("Cluster found: ARN = %v\n", clusterArn)

	// get the cluster
	listTaskInput := ecs.ListTasksInput{
		Cluster: &clusterArn,
	}
	listTaskOutput, err := ecsClient.ListTasks(&listTaskInput)
	if err != nil {
		fmt.Printf("Error listing tasks: %v", err)
		os.Exit(1)
	}
	if len(listTaskOutput.TaskArns) == 0 {
		fmt.Printf("No tasks found")
		os.Exit(0)
	}

	tasks := []*string{}
	for _, t := range listTaskOutput.TaskArns {
		tasks = append(tasks, t)
	}

	describeTaskInput := ecs.DescribeTasksInput{
		Cluster: &clusterArn,
		Tasks:   tasks,
		// if this is not included, the tags associated with the resource are not returned
		Include: []*string{aws.String("TAGS")},
	}

	describeTaskOutput, err := ecsClient.DescribeTasks(&describeTaskInput)

	if err != nil {
		fmt.Printf("Error retrieving task info: %v\n", err.Error())
		os.Exit(1)
	}

	tab := table.NewWriter()
	tab.SetOutputMirror(os.Stdout)
	tab.AppendHeader(table.Row{"task-id", "Project", "Container Image", "Running Time"})

	for _, t := range describeTaskOutput.Tasks {
		taskIDTag := getEcsTag(t.Tags, "gltr-task-id")
		if taskIDTag == nil {
			fmt.Printf("Task has no task ID - ignoring")
			continue
		}
		tab.AppendRow(
			[]interface{}{
				*taskIDTag.Value,
				*t.Containers[0].Name,
				*t.Containers[0].Image,
				time.Now().Sub(*t.StartedAt).String(),
			},
		)
	}

	tab.Render()
}

func getEc2Tasks(ec2Client *ec2.EC2, projectName string) []*ec2.Instance {

	desribeInstanceInput := ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:gltr-managed"),
				Values: []*string{aws.String("true")},
			},
			{
				Name:   aws.String("tag:gltr-project"),
				Values: []*string{aws.String(projectName)},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running")},
			},
		},
	}

	desribeInstancesOutput, err := ec2Client.DescribeInstances(&desribeInstanceInput)
	if err != nil {
		fmt.Printf("Error listing tasks: %v", err)
		os.Exit(1)
	}
	// found nothing
	if len(desribeInstancesOutput.Reservations) == 0 {
		return nil
	}
	if len(desribeInstancesOutput.Reservations) != 1 {
		fmt.Printf("WARNING: expecting single reservation in EC2 API call - only using first Reservation")
	}

	return desribeInstancesOutput.Reservations[0].Instances

}

func getEcsTag(tags []*ecs.Tag, key string) *ecs.Tag {
	if tags == nil {
		fmt.Printf("no tags.")
		return nil
	}
	for _, t := range tags {
		// fmt.Printf("t = %v", *t)
		if *t.Key == key {
			return t
		}
	}

	// if we get to here, we have not found it
	return nil
}

func getEc2Tag(tags []*ec2.Tag, key string) *ec2.Tag {
	for _, t := range tags {
		if *t.Key == key {
			return t
		}
	}

	// if we get to here, we have not found it
	return nil
}

// performs a run on AWS. Assumes the following:
// - AWS credenials are available
// - AWS has been initialized as described elswhere
func ListTasksEc2(projectName string) {

	_, ec2Client, err := getEc2Client()
	if err != nil {
		fmt.Printf("Error initializing AWS session: %v\n", err)
		os.Exit(1)
	}

	tasks := getEc2Tasks(ec2Client, projectName)
	if tasks == nil {
		fmt.Printf("No tasks found.")
		return
	}

	tab := table.NewWriter()
	tab.SetOutputMirror(os.Stdout)
	tab.AppendHeader(table.Row{"task-id", "Project", "Ec2 Instance Type", "Running Time"})

	for _, t := range tasks {
		taskIDTag := getEc2Tag(t.Tags, "gltr-task-id")
		if taskIDTag == nil {
			fmt.Printf("WARNING: cannot find taskID for instance: ignoring")
			continue
		}
		taskID := taskIDTag.Value
		tab.AppendRow(
			[]interface{}{
				*taskID,
				projectName,
				*t.InstanceType,
				time.Now().Sub(*t.LaunchTime).String(),
			},
		)
	}

	tab.Render()
}
