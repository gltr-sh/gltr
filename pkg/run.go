package gltr

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/google/uuid"
	"github.com/oriser/regroup"
	"github.com/pterm/pterm"
	"github.com/samber/lo"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"google.golang.org/api/option"
	computepb "google.golang.org/genproto/googleapis/cloud/compute/v1"
	"google.golang.org/protobuf/proto"
)

var (
	FargateOptionsByCpu = map[int]([]int){

		// comments are taken from (current) AWS comments in the Golang SDK lib - they may
		// change with time.
		//    * 256 (.25 vCPU) - Available memory values: 512 (0.5 GB), 1024 (1 GB),
		//    2048 (2 GB)
		256: []int{512, 1024, 2048},
		//    * 512 (.5 vCPU) - Available memory values: 1024 (1 GB), 2048 (2 GB), 3072
		//    (3 GB), 4096 (4 GB)
		512: []int{1024, 2048, 3072, 4096},
		//    * 1024 (1 vCPU) - Available memory values: 2048 (2 GB), 3072 (3 GB), 4096
		//    (4 GB), 5120 (5 GB), 6144 (6 GB), 7168 (7 GB), 8192 (8 GB)
		1024: []int{2048, 3072, 4096, 5120, 6144, 7168, 8192},
		//    * 2048 (2 vCPU) - Available memory values: 4096 (4 GB) and 16384 (16 GB)
		//    in increments of 1024 (1 GB)
		2048: []int{4096, 5120, 6144, 7168, 8192, 9216, 10240, 11264, 12286, 13312, 14336, 15360, 16384},
		//    * 4096 (4 vCPU) - Available memory values: 8192 (8 GB) and 30720 (30 GB)
		//    in increments of 1024 (1 GB)
		4096: []int{
			8192, 9216, 10240, 11264, 12288, 13312, 14336, 15360, 16384, 17408,
			18432, 19456, 20480, 21504, 22528, 23552, 24576, 25600, 26624, 27648,
			28672, 29696, 30720,
		},
		//    * 8192 (8 vCPU) - Available memory values: 16 GB and 60 GB in 4 GB increments
		//    This option requires Linux platform 1.4.0 or later.
		8192: []int{
			16384, 20480, 24576, 28672, 32768, 36864, 40960, 45056, 49152, 53248, 57344, 61440,
		},
		//    * 16384 (16vCPU) - Available memory values: 32GB and 120 GB in 8 GB increments
		//    This option requires Linux platform 1.4.0 or later.
		16384: []int{
			32768, 40960, 49152, 57344, 65536, 73728, 81920, 90112, 98304, 106496, 114688, 122880, 131072,
		},
	}
)

// Fargate imposes some restrictions on what combinations of CPU and
// memory values are allowed; this performs a validity check and returns
// an error if the combination is not in the allowed set.
func checkCPUMemoryValues(cpu int, mem int) error {
	memValues, exists := FargateOptionsByCpu[cpu]
	if !exists {
		fmt.Printf("Cpu value = %v\n", cpu)
		return errors.New("Invalid CPU option: valid CPU options are [256, 512, 1024, 2048, 4096, 8192, 16384]")
	}

	found := false
	for _, m := range memValues {
		if mem == m {
			found = true
			break
		}
	}
	if !found {
		errorString := fmt.Sprintf(
			"Invalid Memory option for this CPU option (%v) - valid values are %v",
			cpu,
			memValues,
		)
		return errors.New(errorString)
	}
	return nil
}

func generateTaskID() (taskID string) {
	taskID = uuid.New().String()
	return
}

// performs a run on AWS. Assumes the following:
// - AWS credenials are available
// - AWS has been initialized as described elswhere
func RunAwsEcs(gt Task, config Config, gltrPrivateKey []byte, hostname string) (networkAddress string, err error) {

	ecsProjectConfig := gt.GetExecutionPlatformProjectConfig(EcsFargate).(EcsProjectConfig)
	// fmt.Printf("Ecs confg = %v\n", ecsProjectConfig)
	// if ecsConfig == nil {
	// 	fmt.Printf("No ECS configuration found - exiting...\n")
	// 	os.Exit(1)
	// }

	// initialize AWS session
	awsSession, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})

	taskID := generateTaskID()

	pterm.Info.Printf("Initializing communication with AWS\n")
	if err != nil {
		pterm.Error.Printf("Error initializing AWS session: %v\n", err)
		os.Exit(1)
	}

	ecsClient := ecs.New(awsSession)

	pterm.Info.Printf("Obtaining ECS Cluster information\n")
	cluster, err := getCluster(ecsClient, ecsProjectConfig.ClusterName)
	if err != nil {
		fmt.Printf("Error obtaining cluster info: %v\n", err)
		os.Exit(1)
	}

	clusterArn := *cluster.ClusterArn
	pterm.Success.Printf("ECS Cluster found (ARN: %v)\n", clusterArn)

	if err := checkCPUMemoryValues(ecsProjectConfig.CPURequirements, ecsProjectConfig.MemoryRequirements); err != nil {
		pterm.Error.Printf("Invalid cpu/memory values for Fargate: %v", err)
		os.Exit(1)
	}

	b64EncodedPrivateKey := base64.StdEncoding.EncodeToString(gltrPrivateKey)

	user := gt.Users[0]
	b64EncodedSSHKey := base64.StdEncoding.EncodeToString([]byte(user.SshKey))
	b64EncodedUserName := base64.StdEncoding.EncodeToString([]byte(config.User.Name))
	b64EncodedUserEmail := base64.StdEncoding.EncodeToString([]byte(config.User.Email))

	repoFetch, repoPush := getFetchAndPushRepos(gt.GitRepo)

	pterm.Info.Printf("Registering updated task definition\n")
	taskDefinitionInput := ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				Cpu:        lo.ToPtr(int64(ecsProjectConfig.CPURequirements)),
				EntryPoint: []*string{lo.ToPtr("/init")},
				Environment: []*ecs.KeyValuePair{
					{Name: aws.String("SSH_PUBLIC_KEY"), Value: aws.String(b64EncodedSSHKey)},
					{Name: aws.String("GIT_REPO_FETCH"), Value: aws.String(repoFetch)},
					{Name: aws.String("GIT_REPO_PUSH"), Value: aws.String(repoPush)},
					{Name: aws.String("GLTR_PRIVATE_KEY"), Value: aws.String(b64EncodedPrivateKey)},
					{Name: aws.String("GLTR_PROJECT_ID"), Value: aws.String(gt.ProjectID)},
					{Name: aws.String("GLTR_PROJECT_NAME"), Value: aws.String(gt.ProjectName)},
					{Name: aws.String("GLTR_USER_NAME"), Value: aws.String(b64EncodedUserName)},
					{Name: aws.String("GLTR_USER_EMAIL"), Value: aws.String(b64EncodedUserEmail)},
				},
				Image:       aws.String(gt.ContainerImage),
				Interactive: aws.Bool(false),
				Memory:      aws.Int64(int64(ecsProjectConfig.MemoryRequirements)),
				Name:        aws.String(gt.ProjectName),
				// hostname is not supported on fargate with this
				// Hostname:    aws.String(hostname),
				// LogConfiguration:
				// awsvpc only allows exposing ports such that the container port
				// number is the same as the host port number when using FARGATE
				PortMappings: []*ecs.PortMapping{
					{
						ContainerPort: aws.Int64(int64(22)),
						HostPort:      aws.Int64(int64(22)),
					},
					{
						ContainerPort: aws.Int64(int64(8888)),
						HostPort:      aws.Int64(int64(8888)),
					},
				},
			},
		},
		Cpu:                     aws.String(fmt.Sprintf("%v", ecsProjectConfig.CPURequirements)),
		Memory:                  aws.String(fmt.Sprintf("%v", ecsProjectConfig.MemoryRequirements)),
		NetworkMode:             aws.String("awsvpc"),
		RequiresCompatibilities: []*string{aws.String("FARGATE"), aws.String("EC2")},
		// RuntimePlatform: &ecs.RuntimePlatform{
		// 	CpuArchitecture:       lo.ToPtr("x86_64"),
		// 	OperatingSystemFamily: lo.ToPtr("linux"),
		// },
		Tags: []*ecs.Tag{
			{
				Key:   aws.String("gltr-managed"),
				Value: aws.String("true"),
			},
			{
				Key:   aws.String("gltr-project"),
				Value: aws.String(gt.ProjectName),
			},
			{
				Key:   aws.String("gltr-task-id"),
				Value: aws.String(taskID),
			},
		},
		Family: lo.ToPtr("gltr-task"),
	}
	registerTaskDefinitionOutput, err := ecsClient.RegisterTaskDefinition(&taskDefinitionInput)
	if err != nil {
		pterm.Error.Printf("Error registering task: %v\n", err.Error())
		os.Exit(1)
	} else {
		pterm.Success.Printf("Task definition registered (ARN: %v)\n", *registerTaskDefinitionOutput.TaskDefinition.TaskDefinitionArn)
	}

	pterm.Info.Printf("Running task on ECS Cluster\n")
	runTaskInput := ecs.RunTaskInput{
		LaunchType:     lo.ToPtr("FARGATE"),
		TaskDefinition: registerTaskDefinitionOutput.TaskDefinition.TaskDefinitionArn,
		Cluster:        lo.ToPtr(clusterArn),
		Count:          lo.ToPtr(int64(1)),
		NetworkConfiguration: &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
				AssignPublicIp: lo.ToPtr("ENABLED"),
				Subnets: []*string{
					lo.ToPtr(ecsProjectConfig.SubnetID),
				},
				SecurityGroups: []*string{lo.ToPtr(ecsProjectConfig.SecurityGroupID)},
			},
		},
		Tags: []*ecs.Tag{
			{
				Key:   aws.String("gltr-managed"),
				Value: aws.String("true"),
			},
			{
				Key:   aws.String("gltr-project"),
				Value: aws.String(gt.ProjectName),
			},
			{
				Key:   aws.String("gltr-task-id"),
				Value: aws.String(taskID),
			},
		},
		//PropagateTags: aws.String("NONE"),
		// EnableECSManagedTags: aws.Bool(false),
	}

	runTaskOutput, err := ecsClient.RunTask(&runTaskInput)
	if err != nil {
		fmt.Printf("Error registering task: %v\n", err.Error())
		os.Exit(1)
	}

	taskArn := *runTaskOutput.Tasks[0].Containers[0].TaskArn
	spinner, err := pterm.DefaultSpinner.Start("Waiting for task to enter RUNNING state...")

	describeTaskInput := ecs.DescribeTasksInput{
		Cluster: lo.ToPtr(clusterArn),
		Tasks:   []*string{lo.ToPtr(taskArn)},
	}

	startTime := time.Now()
	endTime := startTime.Add(2 * time.Minute)
	running := false
	var describeTaskOutput *ecs.DescribeTasksOutput
	for time.Now().Before(endTime) && running == false {
		describeTaskOutput, err = ecsClient.DescribeTasks(&describeTaskInput)
		if err != nil {
			pterm.Error.Printf("Error retrieving task info: %v\n", err.Error())
			os.Exit(1)
		}
		taskStatus := *describeTaskOutput.Tasks[0].Containers[0].LastStatus
		if taskStatus == "RUNNING" {
			running = true
			spinner.Success("Task entered RUNNING state")
			break
		}
		time.Sleep(10 * time.Second)
	}
	if !running {
		spinner.Fail("Timed out waiting for task to enter RUNNING state")

		// should perform some clean up in this case...
		return "", errors.New("Error waiting for task to enter running state")
	}

	// get eni-id
	networkAddress, err = getNetworkAddressEcs(awsSession, describeTaskOutput.Tasks[0].Attachments[0].Details)

	pterm.Info.Printf("Container IP address: %v\n", networkAddress)
	return
}

func getNetworkAddressEcs(awsSession *session.Session, attachmentDetails []*ecs.KeyValuePair) (string, error) {
	var eniID *string
	for _, n := range attachmentDetails {
		if *n.Name == "networkInterfaceId" {
			eniID = n.Value
			break
		}
	}
	if eniID == nil {
		fmt.Printf("Unable to find Elastic Network Interface ID\n")
		os.Exit(1)
	}

	// now we have the eni id, now we need to convert to a public IP
	ec2Client := ec2.New(awsSession)
	describeNetworkInterfacesInput := ec2.DescribeNetworkInterfacesInput{NetworkInterfaceIds: []*string{eniID}}
	networkInterfaces, err := ec2Client.DescribeNetworkInterfaces(&describeNetworkInterfacesInput)
	if err != nil {
		fmt.Printf("Error retrieving network interfaces: %v\n", err)
		os.Exit(1)
	}
	publicIP := networkInterfaces.NetworkInterfaces[0].Association.PublicDnsName
	if *publicIP == "" {
		publicIP = networkInterfaces.NetworkInterfaces[0].Association.PublicIp
	}
	return *publicIP, nil
}

func launchEc2Instance(ec2Config Ec2ProjectConfig, gt Task) (instanceID, publicDNSName string, err error) {

	pterm.Info.Printf("Initializing communication with AWS\n")

	_, ec2Client, err := getEc2Client()
	if err != nil {
		pterm.Error.Printf("Error initializing EC2 API: %v", err)
		os.Exit(1)
	}

	taskID := generateTaskID()

	// Specify the details of the instance that you want to create.

	pterm.Info.Printf("Launching instance on Ec2...\n")
	runInstancesInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(ec2Config.DefaultImage),
		InstanceType: aws.String(ec2Config.DefaultInstanceType),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		KeyName:      aws.String(ec2Config.KeyName),
		BlockDeviceMappings: []*ec2.BlockDeviceMapping{
			{
				// this is a hardcoded assumption which relates to the
				// way the AMI is created. This really needs to be made
				// more flexible/robust - it's prob possible to query
				// the AMI before creation and get the device identifier
				// of the root disk
				// also, 10GB is hard coded here for no specific reason
				DeviceName: aws.String("/dev/sda1"),
				Ebs: &ec2.EbsBlockDevice{
					VolumeSize: aws.Int64(40),
					// by default, delete the volume on termination...
					DeleteOnTermination: aws.Bool(true),
				},
			},
		},
		// DeleteOnTermination: aws.Bool(true),
		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
			{
				SubnetId:                 aws.String(ec2Config.SubnetID),
				AssociatePublicIpAddress: aws.Bool(true),
				DeviceIndex:              aws.Int64(0),
				Groups: []*string{
					aws.String(ec2Config.SecurityGroupID),
				},
			},
		},
	}
	runInstancesOutput, err := ec2Client.RunInstances(runInstancesInput)

	if err != nil {
		pterm.Error.Println("Error creating Ec2 instance", err)
		return
	}

	instanceID = *runInstancesOutput.Instances[0].InstanceId
	pterm.Info.Printf("Ec2 instance created (id: %v)n", instanceID)

	// Add tags to the created instance
	_, errtag := ec2Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{runInstancesOutput.Instances[0].InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(fmt.Sprintf("gltr Instance (%v)", gt.ProjectName)),
			},
			{
				Key:   aws.String("gltr-managed"),
				Value: aws.String("true"),
			},
			{
				Key:   aws.String("gltr-project"),
				Value: aws.String(gt.ProjectName),
			},
			{
				Key:   aws.String("gltr-task-id"),
				Value: aws.String(taskID),
			},
		},
	})
	if errtag != nil {
		pterm.Error.Println("Could not create tags for instance", runInstancesOutput.Instances[0].InstanceId, errtag)
		return
	}

	// wait for the instance to enter running state...
	// describeInstancesInput := ec2Client.DescribeInstancesInput{}
	describeInstancesInput := ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	}

	startTime := time.Now()
	endTime := startTime.Add(2 * time.Minute)
	running := false
	spinner, err := pterm.DefaultSpinner.Start("Waiting for instance to reach RUNNING state...")
	var describeInstancesOutput *ec2.DescribeInstancesOutput
	for time.Now().Unix() < endTime.Unix() {
		describeInstancesOutput, err = ec2Client.DescribeInstances(&describeInstancesInput)
		if err != nil {
			pterm.Error.Printf("Error obtaining instance info - ignoring...\n")
			continue
		}
		// 16 is the AWS code for running
		if describeInstancesOutput.Reservations[0].Instances[0].State != nil &&
			*describeInstancesOutput.Reservations[0].Instances[0].State.Code == 16 {
			successString := fmt.Sprintf("Instance in RUNNING state after %v", time.Now().Sub(startTime))
			spinner.Success(successString)
			running = true
			break
		}
		time.Sleep(10 * time.Second)
	}

	if !running {
		pterm.Error.Printf(
			"Instance has not reach RUNNING state within 2 minutes...terminating - please check your EC2 account\n",
		)
		os.Exit(1)
	}

	publicDNSName = *describeInstancesOutput.Reservations[0].Instances[0].PublicDnsName
	pterm.Info.Printf("Instance public DNS: %v\n", publicDNSName)

	return

}

func waitForSSH(serverName string, gt Task, port int) (*ssh.Client, error) {
	// var hostKey ssh.PublicKey
	// An SSH client is represented with a ClientConn.
	//
	// To authenticate with the remote server you must pass at least one
	// implementation of AuthMethod via the Auth field in ClientConfig,
	// and provide a HostKeyCallback.
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		log.Fatal(err)
	}
	// defer conn.Close()
	ag := agent.NewClient(conn)
	auths := []ssh.AuthMethod{ssh.PublicKeysCallback(ag.Signers)}

	// this is a terrible hack...
	ec2Config := gt.GetExecutionPlatformProjectConfig(Ec2).(Ec2ProjectConfig)
	var user string
	// not cool at all - manageable for a demo context...
	if ec2Config.GpuRequired {
		user = "ubuntu"
	} else {
		user = "root"
	}
	config := &ssh.ClientConfig{
		User: user,
		// Auth: []ssh.AuthMethod{
		// 	ssh.Password("yourpassword"),
		// },
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	serverWithPort := fmt.Sprintf("%v:%v", serverName, port)
	// dial 10 times with a 10 second delay...
	startTime := time.Now()
	endTime := startTime.Add(2 * time.Minute)
	var client *ssh.Client
	for time.Now().Unix() < endTime.Unix() {
		client, err = ssh.Dial("tcp", serverWithPort, config)
		if err == nil {
			// successful connection established...
			return client, nil
		}
		time.Sleep(10 * time.Second)
	}

	return client, nil
}

func RunAwsEc2(gt Task, config Config, privateKey []byte, hostname string) (publicDnsName string, err error) {
	// fmt.Printf("Ec2 not yet supported.")
	// os.Exit(1)

	// instanceID, publicDnsName, err := launchEc2Instance(gt)
	ec2Config := gt.GetExecutionPlatformProjectConfig(Ec2).(Ec2ProjectConfig)
	_, publicDnsName, err = launchEc2Instance(ec2Config, gt)
	if err != nil {
		fmt.Printf("Error launching EC2 instance: %v\n", err)
		os.Exit(1)
	}

	spinner, err := pterm.DefaultSpinner.Start("Waiting for SSH server to come up...")
	// wait until sshd is running
	client, err := waitForSSH(publicDnsName, gt, 2222)
	if err != nil {
		errorString := fmt.Sprintf("Error establishing ssh connection: %v\n", err)
		spinner.Fail(errorString)
		os.Exit(1)
	}

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := client.NewSession()
	if err != nil {
		errorString := fmt.Sprintf("Failed to create session: %v", err)
		spinner.Fail(errorString)
		os.Exit(1)
	}
	defer session.Close()
	spinner.Success("SSH connection established")

	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	pterm.Info.Printf("Launching docker container inside EC2 instance\n")
	var b bytes.Buffer
	session.Stdout = &b
	taskID := generateTaskID()
	commandArray := createDockerRunInstruction(gt, config, privateKey, taskID, false, hostname, ec2Config.GpuRequired)
	dockerRunString := ""
	for _, c := range commandArray {
		dockerRunString = dockerRunString + c + " "
	}
	// fmt.Printf("Running command: %v\n", dockerRunString)

	if err := session.Run(dockerRunString); err != nil {
		pterm.Error.Printf("Failed to run: %v - retrying", err.Error())
		time.Sleep(10 * time.Second)
		if err := session.Run(dockerRunString); err != nil {
			pterm.Error.Printf("Giving up after second attempt...\n")
			os.Exit(1)
		}
	}
	// if we get here, the run command did not generate an error...
	pterm.Success.Printf("Container launched on ec2 instance\n")

	// fmt.Println(b.String())
	return
}

// this is not robust...the basic idea is ok but the regexes are not well thought through...
func getFetchAndPushRepos(repoName string) (string, string) {
	httpPrefix := "https://github.com"
	gitPrefix := "git@github.com"
	switch {
	case repoName[0:len(httpPrefix)] == httpPrefix:
		fetchURL := repoName
		// convert from a fetchurl to a git url
		// FIXME - this is not robust but in fact it is prob not used frequently, so
		// it was deprioritized
		re := regroup.MustCompile("https://github.com/(?P<name>\\w+)/(?P<repo>.+)")
		matches, _ := re.Groups(repoName)
		name := matches["name"]
		repo := matches["repo"]
		pushURL := fmt.Sprintf("git@github.com:%s/%s", name, repo)
		return fetchURL, pushURL
	case repoName[0:len(gitPrefix)] == gitPrefix:
		pushURL := repoName
		// convert from a git url to a http url
		// original taken from Stack Overflow - covers all scenarios
		// ((git|ssh|http(s)?)|(git@[\w\.-]+))(:(//)?)([\w\.@\:/\-~]+)(\.git)(/)?
		re := regroup.MustCompile("git@(?P<gitprovider>[\\w\\.-]+):(?P<repo>[\\w\\.@\\:/\\-~]+)(\\.git)(/)?")
		matches, _ := re.Groups(repoName)
		gitprovider := matches["gitprovider"]
		repo := matches["repo"]
		fetchURL := fmt.Sprintf("https://%s/%s.git", gitprovider, repo)
		return fetchURL, pushURL
	default:
		fmt.Printf("Warning - cannot parse fetch and push repos\n")
	}
	return repoName, repoName
}

func createDockerRunInstruction(
	gt Task,
	config Config,
	privateKey []byte,
	taskID string,
	dynamicPortAssignment bool,
	hostname string,
	useGpus bool,
) (command []string) {

	b64EncodedPrivateKey := base64.StdEncoding.EncodeToString(privateKey)
	user := gt.Users[0]
	b64EncodedSSHKey := base64.StdEncoding.EncodeToString([]byte(user.SshKey))
	b64EncodedUserName := base64.StdEncoding.EncodeToString([]byte(config.User.Name))
	b64EncodedUserEmail := base64.StdEncoding.EncodeToString([]byte(config.User.Email))

	repoFetch, repoPush := getFetchAndPushRepos(gt.GitRepo)

	// build the command...
	command = append(command, "docker", "run", "--rm", "-d")
	// // the pubkey contains spaces, so we need the quotes here...
	envVar := fmt.Sprintf("SSH_PUBLIC_KEY=%v", b64EncodedSSHKey)
	command = append(command, "-e", envVar)
	envVar = fmt.Sprintf("GIT_REPO_FETCH=%v", repoFetch)
	command = append(command, "-e", envVar)
	envVar = fmt.Sprintf("GIT_REPO_PUSH=%v", repoPush)
	command = append(command, "-e", envVar)
	envVar = fmt.Sprintf("GLTR_PRIVATE_KEY=%v", b64EncodedPrivateKey)
	command = append(command, "-e", envVar)
	envVar = fmt.Sprintf("GLTR_PROJECT_ID=%v", gt.ProjectID)
	command = append(command, "-e", envVar)
	envVar = fmt.Sprintf("GLTR_PROJECT_NAME=%v", gt.ProjectName)
	command = append(command, "-e", envVar)
	envVar = fmt.Sprintf("GLTR_USER_NAME=%v", b64EncodedUserName)
	command = append(command, "-e", envVar)
	envVar = fmt.Sprintf("GLTR_USER_EMAIL=%v", b64EncodedUserEmail)
	command = append(command, "-e", envVar)
	if useGpus {
		command = append(command, "--gpus", "all")
	}
	command = append(command, "-l", "gltr-managed=true")
	envVar = fmt.Sprintf("gltr-task-id=%v", taskID)
	command = append(command, "-l", envVar)
	command = append(command, "--hostname", hostname)
	if dynamicPortAssignment {
		// open ports, but we will need to determine wihch ports on the local
		// machine have been bound
		command = append(command, "-p", "8888", "-p", "22")
	} else {
		command = append(command, "-p", "8888:8888", "-p", "22:22")
	}
	command = append(command, "--name", gt.ProjectName)
	command = append(command, gt.ContainerImage)

	return
}

// gcloud compute instances create instance-1 \
//   --project=$PROJECT \
//   --zone=us-central1-a \
//   --machine-type=n1-standard-1 \
//   --network-interface=network-tier=PREMIUM,subnet=default \
//   --metadata=ssh-keys=sean:ssh-rsa\ AAAAB3NzaC1yc2EAAAADAQABAAACAQCeUnxYGfGebTcsrRHgX0MDyxXv7zxB0coC8bpj8FUntB798fV6z\+ZS8AT2jfhlk64RUccs/VSFYYyJyLXSyFsAIJReqEMjBK0Gyq63eMFwiqycvWW5n\+tNWH98AlSGurl6aAGhUbL67s12mdF3/xAcyQw/U5hPNxbLuT1g6k0Rn9GCVjA\+\+d76W9xLEYp/rat0Mwp\+Rys7/D4Y0x8FLJxl3OqtW\+Q5BNDTR0BRV17YtVhchiWwQI063nD/l/3\+Vuuw7HvgoKskGTLxEruqoACgq16ld6MFHZMX0HcgyEfMkAXwMC5RhFIA0wS\+ueGe5GWPj6UQklEpL1PCh8V5uqVrSNkTkCH8FaK9Z0p4rC43euUalOgm8ujPrvJbiWCoQbtZwR3MpD74pgGBt4HcJlDv2LKwnSo11jB\+BUR0zPZO99UBuTDcLtJRfCbKLhnPnuLbRPRD4tEHfFnHjJqG6bRD9hRoQVM0s5TKPQJfH3ZOtmRi6FPXmVabLxfFbxqYQFmalLZHc63UApGUeKogxPTQoRPn1nros13JM5F9CVwe2uTELUWbDLkP8oJHPhd3nDstu88qpdCcYNBcgSrxU36YvtcV3ABIaxUFqFL49rqz9J7dnSI4uXeoLHGlv8qe0k9BeD2G3K0QWRHpWZ7VxMaxfHXIvOM6i7ddWtkCNU7EvQ==\ sean \
//   --maintenance-policy=TERMINATE \
//   --provisioning-model=STANDARD \
//   --service-account=437447506792-compute@developer.gserviceaccount.com \
//   --scopes=https://www.googleapis.com/auth/devstorage.read_only,https://www.googleapis.com/auth/logging.write,https://www.googleapis.com/auth/monitoring.write,https://www.googleapis.com/auth/servicecontrol,https://www.googleapis.com/auth/service.management.readonly,https://www.googleapis.com/auth/trace.append \
//   --accelerator=count=1,type=nvidia-tesla-t4 \
//   --create-disk=auto-delete=yes,boot=yes,device-name=instance-1,image=projects/ml-images/global/images/c0-deeplearning-common-cpu-v20230126-debian-10,mode=rw,size=50,type=projects/gltr-testing/zones/us-central1-a/diskTypes/pd-balanced \
//   --no-shielded-secure-boot \
//   --shielded-vtpm \
//   --shielded-integrity-monitoring \
//   --reservation-affinity=any

// TODO: createInstanceGcp has not operational right now
// createInstance sends an instance creation request to the Compute Engine API and waits for it to complete.
func createInstanceGcp(
	w io.Writer,
	projectID, zone, instanceName, machineType, sourceImage, networkName string,
) (string, string, error) {
	// zone := "us-central1-a"
	// instanceName := "gltr-instance"
	// machineType := "n1-standard-1"
	// sourceImage := "projects/debian-cloud/global/images/family/debian-10"
	// networkName := "global/networks/default"

	log.Printf("Launching VM on GCP")
	ctx := context.Background()
	// clearly need a more sensible way to pick up GCP credentials
	clientOption := option.WithCredentialsFile("/home/sean/.config/gcloud/legacy_credentials/sean@gopaddy.ch/adc.json")
	instancesClient, err := compute.NewInstancesRESTClient(ctx, clientOption)
	if err != nil {
		return "", "", fmt.Errorf("NewInstancesRESTClient: %v", err)
	}
	defer instancesClient.Close()

	req := &computepb.InsertInstanceRequest{
		Project: projectID,
		Zone:    zone,
		InstanceResource: &computepb.Instance{
			Name: proto.String(instanceName),
			Disks: []*computepb.AttachedDisk{
				{
					InitializeParams: &computepb.AttachedDiskInitializeParams{
						DiskSizeGb:  proto.Int64(50),
						SourceImage: proto.String(sourceImage),
					},
					AutoDelete: proto.Bool(true),
					Boot:       proto.Bool(true),
					Type:       proto.String(computepb.AttachedDisk_PERSISTENT.String()),
				},
			},
			MachineType: proto.String(fmt.Sprintf("zones/%s/machineTypes/%s", zone, machineType)),
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					Name: proto.String(networkName),
					// this configuration assigned a public IP address...
					AccessConfigs: []*computepb.AccessConfig{
						{
							Type: proto.String("ONE_TO_ONE_NAT"),
							Name: proto.String("External NAT"),
						},
					},
				},
			},
			Metadata: &computepb.Metadata{
				Items: []*computepb.Items{
					{
						Key: proto.String("ssh-keys"),
						Value: proto.String(
							"sean:ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCeUnxYGfGebTcsrRHgX0MDyxXv7zxB0coC8bpj8FUntB798fV6z+ZS8AT2jfhlk64RUccs/VSFYYyJyLXSyFsAIJReqEMjBK0Gyq63eMFwiqycvWW5n+tNWH98AlSGurl6aAGhUbL67s12mdF3/xAcyQw/U5hPNxbLuT1g6k0Rn9GCVjA++d76W9xLEYp/rat0Mwp+Rys7/D4Y0x8FLJxl3OqtW+Q5BNDTR0BRV17YtVhchiWwQI063nD/l/3+Vuuw7HvgoKskGTLxEruqoACgq16ld6MFHZMX0HcgyEfMkAXwMC5RhFIA0wS+ueGe5GWPj6UQklEpL1PCh8V5uqVrSNkTkCH8FaK9Z0p4rC43euUalOgm8ujPrvJbiWCoQbtZwR3MpD74pgGBt4HcJlDv2LKwnSo11jB+BUR0zPZO99UBuTDcLtJRfCbKLhnPnuLbRPRD4tEHfFnHjJqG6bRD9hRoQVM0s5TKPQJfH3ZOtmRi6FPXmVabLxfFbxqYQFmalLZHc63UApGUeKogxPTQoRPn1nros13JM5F9CVwe2uTELUWbDLkP8oJHPhd3nDstu88qpdCcYNBcgSrxU36YvtcV3ABIaxUFqFL49rqz9J7dnSI4uXeoLHGlv8qe0k9BeD2G3K0QWRHpWZ7VxMaxfHXIvOM6i7ddWtkCNU7EvQ== sean",
						),
					},
				},
			},
		},
	}

	op, err := instancesClient.Insert(ctx, req)
	if err != nil {
		return "", "", fmt.Errorf("unable to create instance: %v", err)
	}

	if err = op.Wait(ctx); err != nil {
		return "", "", fmt.Errorf("unable to wait for the operation: %v", err)
	}

	// instance has been created - it seems we need to make another query to get
	// information on the instance
	getRequest := &computepb.GetInstanceRequest{
		Project:  projectID,
		Zone:     zone,
		Instance: instanceName,
	}
	instance, err := instancesClient.Get(ctx, getRequest)
	if err != nil {
		return "", "", fmt.Errorf("unable to get instance: %v", err)
	}

	instanceID := *instance.Id
	instanceIPAddress := *instance.NetworkInterfaces[0].AccessConfigs[0].NatIP

	log.Printf("Instance created (Id %v) with IP address %v\n", instanceID, instanceIPAddress)

	return string(instanceID), instanceIPAddress, nil
}

// this is currently not used....
func RunGcp(gt Task, config Config, gltrPrivateKey []byte) error {

	// launch VM
	projectID := "gltr-testing"
	zone := "us-central1-a"
	instanceName := "gltr-instance"
	machineType := "n1-standard-1"
	sourceImage := "projects/ml-images/global/images/c0-deeplearning-common-cpu-v20230126-debian-10"
	networkName := "global/networks/default"

	var buf bytes.Buffer
	_, publicDNSName, err := createInstanceGcp(
		&buf,
		projectID,
		zone,
		instanceName,
		machineType,
		sourceImage,
		networkName,
	)
	if err != nil {
		fmt.Printf("Error creating GCP instance: %s\n", err)
		return err
	}

	client, err := waitForSSH(publicDNSName, gt, 22)
	// log.Printf("conn = %v %v\n", conn, ag)
	if err != nil {
		fmt.Printf("Error establishing ssh connection: %v\n", err)
		return err
	}

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := client.NewSession()
	if err != nil {
		fmt.Printf("Failed to create session: %v", err)
		os.Exit(1)
	}
	defer session.Close()

	// Once a Session is created, you can execute a single command on
	// the remote side using the Run method.
	var b bytes.Buffer
	session.Stdout = &b
	taskID := generateTaskID()
	commandArray := createDockerRunInstruction(gt, config, gltrPrivateKey, taskID, true, "gcp-testing", false)
	dockerRunString := ""
	for _, c := range commandArray {
		dockerRunString = dockerRunString + c + " "
	}
	fmt.Printf("Running command: %v\n", dockerRunString)

	if err := session.Run(dockerRunString); err != nil {
		fmt.Printf("Failed to run: %v - retrying", err.Error())
		time.Sleep(10 * time.Second)
		fmt.Printf("trying again...")
		if err := session.Run(dockerRunString); err != nil {
			fmt.Printf("Giving up after second attempt...\n")
			os.Exit(1)
		}
	}
	fmt.Println(b.String())
	return nil
}
