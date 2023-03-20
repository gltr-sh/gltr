package gltr

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/erikgeiser/promptkit/confirmation"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

func getRoutingTable(svc *ec2.EC2, vpcID string) (routingTable ec2.RouteTable, err error) {
	describeRouteTablesInput := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{{Name: aws.String("vpc-id"), Values: []*string{aws.String(vpcID)}}},
	}
	describeRouteTablesOutput, err := svc.DescribeRouteTables(describeRouteTablesInput)
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

	// log.Printf("Routing table output = %v", describeRouteTablesOutput)

	// each vpc should simply have a single default routing table
	routingTable = *describeRouteTablesOutput.RouteTables[0]
	return
}

func addSecurityGroupRule(svc *ec2.EC2, securityGroupID string, port int) (err error) {
	// add two inbound rules to the secgroup
	secgroupRuleInput := &ec2.AuthorizeSecurityGroupIngressInput{
		CidrIp:     aws.String("0.0.0.0/0"),
		ToPort:     lo.ToPtr(int64(port)),
		FromPort:   lo.ToPtr(int64(port)),
		GroupId:    aws.String(securityGroupID),
		IpProtocol: aws.String("tcp"),
	}
	_, err = svc.AuthorizeSecurityGroupIngress(secgroupRuleInput)
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

func createSecurityGroup(svc *ec2.EC2, vpcID string, name string) (securityGroupID string, err error) {

	createSecurityGroupInput := &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(name),
		VpcId:       &vpcID,
		Description: aws.String("Project specific security group for gltr"),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("security-group"),
				Tags: []*ec2.Tag{
					{Key: aws.String("Name"), Value: aws.String(name)},
					{Key: aws.String("gltr-managed"), Value: aws.String("true")},
				},
			},
		},
	}
	createSecurityGroupOutput, err := svc.CreateSecurityGroup(createSecurityGroupInput)
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
	fmt.Printf(
		"Security group created: id=%v\n", *createSecurityGroupOutput.GroupId)
	securityGroupID = *createSecurityGroupOutput.GroupId

	return
}

func createSubnet(svc *ec2.EC2, vpcID string) (subnetID string, err error) {
	createSubnetInput := &ec2.CreateSubnetInput{
		CidrBlock: aws.String("10.0.1.0/24"),
		VpcId:     aws.String(vpcID),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("subnet"),
				Tags: []*ec2.Tag{
					{Key: lo.ToPtr("Name"), Value: lo.ToPtr("gltr-subnet")},
					{Key: lo.ToPtr("gltr-managed"), Value: lo.ToPtr("true")},
				},
			},
		},
	}

	subnet, err := svc.CreateSubnet(createSubnetInput)
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
	subnetID = *subnet.Subnet.SubnetId
	fmt.Printf("Successfully created subnet id=%v\n", subnetID)
	return
}

func createVpc(svc *ec2.EC2) (vpc ec2.Vpc, err error) {
	createVpcInput := &ec2.CreateVpcInput{
		CidrBlock: aws.String("10.0.0.0/16"),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("vpc"),
				Tags: []*ec2.Tag{
					{Key: lo.ToPtr("Name"), Value: lo.ToPtr("gltr-vpc")},
					{Key: lo.ToPtr("gltr-managed"), Value: lo.ToPtr("true")},
				},
			},
		},
	}

	createVpcOutput, err := svc.CreateVpc(createVpcInput)
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

	modifyVpcAttributeInput := &ec2.ModifyVpcAttributeInput{
		EnableDnsHostnames: &ec2.AttributeBooleanValue{Value: aws.Bool(true)},
		VpcId:              createVpcOutput.Vpc.VpcId,
	}
	_, err = svc.ModifyVpcAttribute(modifyVpcAttributeInput)
	if err != nil {
		fmt.Printf("Error modifying VPC attribute: %s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("Sucessfully created VPC id=%v\n", *createVpcOutput.Vpc.VpcId)
	return *createVpcOutput.Vpc, nil
}

func setupNetworking(awsSession *session.Session) (vpcID, igwID, subnetID string, err error) {

	svc := ec2.New(awsSession)

	// create the VPC
	vpc, err := createVpc(svc)
	vpcID = *vpc.VpcId

	createInternetGatewayInput := &ec2.CreateInternetGatewayInput{
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("internet-gateway"),
				Tags: []*ec2.Tag{
					{Key: lo.ToPtr("Name"), Value: lo.ToPtr("gltr-igw")},
					{Key: lo.ToPtr("gltr-managed"), Value: lo.ToPtr("true")},
				},
			},
		},
	}
	createInternetGatewayOutput, err := svc.CreateInternetGateway(createInternetGatewayInput)
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
	fmt.Printf(
		"Successfully create Internet gateway id=%v\n",
		*createInternetGatewayOutput.InternetGateway.InternetGatewayId,
	)
	igwID = *createInternetGatewayOutput.InternetGateway.InternetGatewayId

	attachInternetGatewayInput := &ec2.AttachInternetGatewayInput{
		VpcId:             &vpcID,
		InternetGatewayId: &igwID,
	}
	_, err = svc.AttachInternetGateway(attachInternetGatewayInput)
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
	// log.Printf("igw attached to vpc")

	subnetID, err = createSubnet(svc, vpcID)
	if err != nil {
		fmt.Printf("Error creating subnet: %s", err.Error())
		// have to add cleanup functions here...
		os.Exit(1)

	}

	routingTable, err := getRoutingTable(svc, vpcID)
	routingTableID := *routingTable.RouteTableId

	// add route to subnet
	createRouteInput := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(igwID),
		RouteTableId:         aws.String(routingTableID),
	}
	_, err = svc.CreateRoute(createRouteInput)
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
	// log.Printf("igw route added to subnet")

	return
}

func createEcsCluster(awsSession *session.Session) (cluster *ecs.Cluster, err error) {
	_, ecsClient, err := getEcsClient()
	createClusterInput := &ecs.CreateClusterInput{
		ClusterName: lo.ToPtr("gltr-cluster"),
		// capaciity providers are either autoscaling groups or fargate...
		CapacityProviders: []*string{lo.ToPtr("FARGATE"), lo.ToPtr("FARGATE_SPOT")},
		Tags: []*ecs.Tag{
			{Key: lo.ToPtr("gltr-managed"), Value: lo.ToPtr("true")},
		},
	}
	createClusterOutput, err := ecsClient.CreateCluster(createClusterInput)
	if err != nil {
		log.Printf("Error creating cluster: %v", err.Error())
		return
	}
	cluster = createClusterOutput.Cluster
	return
}

// InitializeAWS needs to create the following:
// - a VPC and subnet which have public connectivity
// - an ECS cluster
// - optionally a cluster role which supports adding cloudwatch to the cluster
func InitializeAWS() (config AWSConfig, err error) {

	// this could be an input parameter here; need to be more precise
	// about the interface
	useDefaultAwsConfiguration := ReadConfirmationInput("Use default AWS configuration", confirmation.Yes)
	if !useDefaultAwsConfiguration {
		fmt.Printf("Not implemented yet - exiting...\n")
		os.Exit(1)
	}
	fmt.Printf("\n")

	awsSession, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})

	if err != nil {
		fmt.Printf("Error initializing AWS session: %v", err)
		return
	}

	// vpcId, subnetId, securityGroupID, err := setupNetworking(awsSession)
	vpcID, igwID, subnetID, err := setupNetworking(awsSession)
	config.SubnetID = subnetID
	config.VpcID = vpcID
	config.IgwID = igwID

	// this is ugly but workable for now...
	fmt.Printf("WARNING: Default AMI image is currently hardcoded...\n")
	// this should be fixed up such that we query AWS for the ami and the keyname
	config.DefaultAmiCPUImage = "ami-0a29e902d4020927e"
	config.DefaultAmiGPUImage = "ami-08f55edd71b55694a"
	config.Initialized = true
	// config.SecurityGroup = securityGroupID
	return

}

func WriteAWSConfig(config AWSConfig, filename string) {
	dat, err := yaml.Marshal(config)
	if err != nil {
		fmt.Printf("Error marshalling AWS config: %s", err)
	}

	err = os.WriteFile(filename, dat, 0644)
	if err != nil {
		fmt.Printf("Error writing AWS config file: %s", err)
	}

	return
}

func ReadAWSConfig(filename string) (config AWSConfig, err error) {
	dat, err := os.ReadFile(filename)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(dat, &config)
	// if err == nil or != nil, we let the caller handle it...
	return
}
