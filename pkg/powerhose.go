package gltr

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func removeCluster(clusterName string) (err error) {
	_, ecsClient, err := getEcsClient()
	deleteClusterInput := ecs.DeleteClusterInput{
		Cluster: &clusterName,
	}

	_, err = ecsClient.DeleteCluster(&deleteClusterInput)
	if err != nil {
		log.Printf("Error deleting cluster: %v", err.Error())
		return
	}
	return
}

func removeVPC(ec2Client *ec2.EC2, vpcID string) (err error) {

	deleteVpcInput := ec2.DeleteVpcInput{VpcId: aws.String(vpcID)}

	_, err = ec2Client.DeleteVpc(&deleteVpcInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	return
}

func removeInternetGateway(ec2Client *ec2.EC2, igwID string) (err error) {
	deleteInternetGatewayInput := ec2.DeleteInternetGatewayInput{InternetGatewayId: aws.String(igwID)}

	_, err = ec2Client.DeleteInternetGateway(&deleteInternetGatewayInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	return
}

func removeSubnet(ec2Client *ec2.EC2, subnetID string) (err error) {
	deleteSubnetInput := ec2.DeleteSubnetInput{SubnetId: aws.String(subnetID)}

	_, err = ec2Client.DeleteSubnet(&deleteSubnetInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	return
}

func removeRoutes(ec2Client *ec2.EC2, vpcID string) (err error) {

	routingTable, err := getRoutingTable(ec2Client, vpcID)
	routingTableID := *routingTable.RouteTableId

	// remove the route to 0.0.0.0
	for _, route := range routingTable.Routes {
		destinationCidrBlock := aws.StringValue(route.DestinationCidrBlock)
		if destinationCidrBlock == "0.0.0.0/0" {
			// fmt.Printf("Removing route to 0.0.0.0/0")
			DeleteRouteInput := ec2.DeleteRouteInput{
				RouteTableId:         aws.String(routingTableID),
				DestinationCidrBlock: route.DestinationCidrBlock,
			}
			_, err = ec2Client.DeleteRoute(&DeleteRouteInput)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					default:
						fmt.Println(aerr.Error())
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					fmt.Println(err.Error())
				}
				return
			}
		}
	}
	return

}

func detachInternetGateway(ec2Client *ec2.EC2, igwID, vpcID string) (err error) {
	detachInternetGatewayInput := ec2.DetachInternetGatewayInput{
		InternetGatewayId: aws.String(igwID),
		VpcId:             aws.String(vpcID),
	}
	_, err = ec2Client.DetachInternetGateway(&detachInternetGatewayInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}
	return
}

func removeSecurityGroups(ec2Client *ec2.EC2, vpcID string) (err error) {
	describeSecurityGroupsInput := ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{{Name: aws.String("vpc-id"), Values: []*string{aws.String(vpcID)}}},
	}

	describeSecurityGroupsOutput, err := ec2Client.DescribeSecurityGroups(&describeSecurityGroupsInput)

	for _, s := range describeSecurityGroupsOutput.SecurityGroups {
		if *s.GroupName != "default" {
			deleteSecurityGroupInput := ec2.DeleteSecurityGroupInput{GroupId: s.GroupId}

			_, err = ec2Client.DeleteSecurityGroup(&deleteSecurityGroupInput)

			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					default:
						fmt.Println(aerr.Error())
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					fmt.Println(err.Error())
				}
				return
			}
		}
	}
	return

}

func removeNetworkConfig(c AWSConfig) (err error) {
	_, ec2Client, err := getEc2Client()

	err = removeSecurityGroups(ec2Client, c.VpcID)
	if err != nil {
		fmt.Printf("Error removing security groups - terminating powerhose operation: %v", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Successfully removed security groups\n")

	// remove the subnet
	err = removeSubnet(ec2Client, c.SubnetID)
	if err != nil {
		fmt.Printf("Error removing subnet - terminating powerhose operation: %v", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Successfully removed subnet %s\n", c.SubnetID)

	// remove routes from the routing table
	err = removeRoutes(ec2Client, c.VpcID)
	if err != nil {
		fmt.Printf("Error removing subnet - terminating powerhose operation: %v", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Successfully removed default route from routing table\n")

	// detach the igw from the vpc
	err = detachInternetGateway(ec2Client, c.IgwID, c.VpcID)
	if err != nil {
		fmt.Printf("Error removing subnet - terminating powerhose operation: %v", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Successfully detached internet gateway %v from vpc %v\n", c.IgwID, c.VpcID)

	// remove the igw
	err = removeInternetGateway(ec2Client, c.IgwID)
	if err != nil {
		fmt.Printf("Error removing internet gateway - terminating powerhose operation: %v", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Successfully removed internet gateway %s\n", c.SubnetID)

	// remove the vpc
	err = removeVPC(ec2Client, c.VpcID)
	if err != nil {
		fmt.Printf("Error removing vpc - terminating powerhose operation: %v", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Successfully removed vpc %s\n", c.VpcID)
	return
}

func PowerhoseAws(c AWSConfig) (err error) {
	fmt.Printf("Powerhosing aws\n")

	clusterName := "test"
	err = removeCluster(clusterName)
	if err != nil {
		fmt.Printf("Terminating powerhose operation...")
		return
	}
	fmt.Printf("Cluster %v removed\n", clusterName)

	err = removeNetworkConfig(c)
	if err != nil {
		fmt.Printf("Terminating powerhose operation...")
		return
	}

	return
}
