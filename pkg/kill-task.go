package gltr

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/samber/lo"
)

func findTaskWithTag(ecsClient *ecs.ECS, clusterArn, taskID string) (taskArn string) {
	// we need to find the task with the given tag...we do this by
	// getting all tasks and filtering
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

	// find the task we want...
	var taskIDTag *ecs.Tag
	for _, t := range describeTaskOutput.Tasks {
		taskIDTag = getEcsTag(t.Tags, "gltr-task-id")
		if taskIDTag != nil {
			if *taskIDTag.Value == taskID {
				taskArn = *t.TaskArn
				break
			}
		}
	}

	if taskArn == "" {
		fmt.Printf("Unable to find task with ID %v\n", taskID)
		os.Exit(1)
	}
	return

}

// performs a run on AWS. Assumes the following:
// - AWS credenials are available
// - AWS has been initialized as described elswhere
func KillTaskEcs(clusterName, taskID string) {

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

	taskArn := findTaskWithTag(ecsClient, clusterArn, taskID)

	stopTaskInput := &ecs.StopTaskInput{
		Cluster: &clusterArn,
		Task:    lo.ToPtr(taskArn),
	}
	stopTaskOutput, err := ecsClient.StopTask(stopTaskInput)
	if err != nil {
		fmt.Printf("Error stopping task: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Task %v killed (ARN %v)\n", taskID, *stopTaskOutput.Task.TaskArn)
}

func KillTaskEc2(taskID string) {

	_, ec2Client, err := getEc2Client()
	if err != nil {
		fmt.Printf("Error initializing AWS session: %v\n", err)
		os.Exit(1)
	}

	desribeInstanceInput := ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag:gltr-task-id"),
				Values: []*string{aws.String(taskID)},
			},
		},
	}

	desribeInstancesOutput, err := ec2Client.DescribeInstances(&desribeInstanceInput)
	if err != nil {
		fmt.Printf("Error listing tasks: %v\n", err)
		return
	}
	// found nothing
	if len(desribeInstancesOutput.Reservations) == 0 {
		fmt.Printf("Error finding task %v\n", taskID)
		return
	}

	if len(desribeInstancesOutput.Reservations) != 1 {
		fmt.Printf("WARNING: expecting single reservation in EC2 API call - only using first Reservation")
	}

	instanceID := desribeInstancesOutput.Reservations[0].Instances[0].InstanceId

	terminateInstancesInput := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{aws.String(*instanceID)},
	}
	terminateInstancesOutput, err := ec2Client.TerminateInstances(terminateInstancesInput)
	if err != nil {
		fmt.Printf("Error terminating instance: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf(
		"Task %v killed (ec2 instance %v terminated)\n",
		taskID,
		*terminateInstancesOutput.TerminatingInstances[0].InstanceId,
	)
}
