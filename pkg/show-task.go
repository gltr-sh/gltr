package gltr

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
)

// performs a run on AWS. Assumes the following:
// - AWS credenials are available
// - AWS has been initialized as described elswhere
func ShowTaskEcs(clusterName, taskID string) {

	awsSession, ecsClient, err := getEcsClient()
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
	// get the cluster
	describeTaskInput := &ecs.DescribeTasksInput{
		Cluster: &clusterArn,
		Tasks:   []*string{lo.ToPtr(taskArn)},
		// Tags: []*ecs.Tag{
		// 	{
		// 		Key:   lo.ToPtr("gltr-managed"),
		// 		Value: lo.ToPtr("true"),
		// 	},
		// },
	}

	describeTaskOutput, err := ecsClient.DescribeTasks(describeTaskInput)
	if err != nil {
		fmt.Printf("Error obtaining task information: %v\n", err)
		os.Exit(1)
	}
	if len(describeTaskOutput.Tasks) != 1 {
		fmt.Printf("Error - task not found\n")
		os.Exit(1)
	}

	// get task ip/name
	networkAddress, err := getNetworkAddressEcs(awsSession, describeTaskOutput.Tasks[0].Attachments[0].Details)

	tab := table.NewWriter()
	tab.SetOutputMirror(os.Stdout)
	tab.AppendHeader(table.Row{"Parameter", "Value"})

	t := describeTaskOutput.Tasks[0]
	tab.AppendRow([]interface{}{"Name", *t.Containers[0].Name})
	tab.AppendRow([]interface{}{"Container Image", *t.Containers[0].Image})
	tab.AppendRow([]interface{}{"Start Time", t.StartedAt.String()})
	tab.AppendRow([]interface{}{"Running Time", time.Now().Sub(*t.StartedAt).String()})
	tab.AppendRow([]interface{}{"CPU", *t.Containers[0].Cpu})
	tab.AppendRow([]interface{}{"Memory", *t.Containers[0].Memory})
	tab.AppendRow([]interface{}{"Network Address", networkAddress})
	tab.AppendRow([]interface{}{"Task Arn", *t.TaskArn})
	tab.AppendRow([]interface{}{"Task ID", taskID})

	tab.Render()
	// here we print out the following: name, container image, start time and
	// running time, cpu, memory and IP addr,
}

// performs a run on AWS. Assumes the following:
// - AWS credenials are available
// - AWS has been initialized as described elswhere
func ShowTaskEc2(taskID string) {

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
		fmt.Printf("WARNING: expecting single reservation in EC2 API call - only using first Reservation\n")
	}

	i := desribeInstancesOutput.Reservations[0].Instances[0]

	tab := table.NewWriter()
	tab.SetOutputMirror(os.Stdout)
	tab.AppendHeader(table.Row{"Parameter", "Value"})

	nameTag := getEc2Tag(i.Tags, "Name")
	var name string
	if nameTag == nil {
		name = "-"
	} else {
		name = *nameTag.Value
	}

	tab.AppendRow([]interface{}{"EC2 Instance Name", name})
	tab.AppendRow([]interface{}{"EC2 Instance Type", *i.InstanceType})
	tab.AppendRow([]interface{}{"Machine Image ID", *i.ImageId})
	tab.AppendRow([]interface{}{"Start Time", i.LaunchTime.String()})
	tab.AppendRow([]interface{}{"Running Time", time.Now().Sub(*i.LaunchTime).String()})
	tab.AppendRow([]interface{}{"Network Address", *i.PublicDnsName})
	tab.AppendRow([]interface{}{"Task-ID", taskID})

	tab.Render()
	// here we print out the following: name, container image, start time and
	// running time, cpu, memory and IP addr,
}
