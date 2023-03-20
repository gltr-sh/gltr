package gltr

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/docker/docker/client"
)

func ConfigAddDockerExecutionPlatform(config Config) (ExecutionPlatform, error) {
	// check if docker engine exists locally - we probably need to check if other
	// container runtimes are support here, eg singularity or podman
	fmt.Printf("Checking for local docker engine...\n")

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("Error creating docker client: %v\n", err)
		return ExecutionPlatform{}, err
	}

	// get info about the docker server...
	info, err := cli.Info(context.Background())
	if err != nil {
		fmt.Printf("Error obtaining information about docker server: %v\n", err)
		return ExecutionPlatform{}, err
	}
	var runtimeArray []string
	for r := range info.Runtimes {
		runtimeArray = append(runtimeArray, r)
	}
	fmt.Printf("Docker engine version %v found (runtimes %v)\n", info.ServerVersion, runtimeArray)

	var runtimes []string
	for r := range info.Runtimes {
		runtimes = append(runtimes, r)
	}

	dockerConfig := DockerConfig{
		Enabled:       true,
		EngineVersion: info.ServerVersion,
		Runtimes:      runtimes,
	}
	executionPlatform := ExecutionPlatform{
		Type:          Docker,
		Configuration: dockerConfig,
	}
	return executionPlatform, nil
}

func getKeyList(session *session.Session) ([]string, error) {
	ec2Client := ec2.New(session)
	describeKeyPairsInput := ec2.DescribeKeyPairsInput{}

	describeKeyPairsOutput, err := ec2Client.DescribeKeyPairs(&describeKeyPairsInput)
	if err != nil {
		return nil, err
	}

	var keypairs []string
	for _, k := range describeKeyPairsOutput.KeyPairs {
		keypairs = append(keypairs, *k.KeyName)
	}
	return keypairs, nil
}

func ConfigAddEc2ExecutionPlatform(config Config) (Config, error) {

	// create session
	awsSession, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		fmt.Printf("Error initializing AWS session: %v", err)
		return Config{}, err
	}

	var awsConfig AWSConfig
	if !config.ProviderConfiguration.AWS.Initialized {
		awsConfig, err = InitializeAWS()
		if err != nil {
			return Config{}, err
		}
		config.ProviderConfiguration.AWS = awsConfig
	}

	// with a valid AWS config, we initialize ECS Fargate
	ec2Config, err := initializeEc2(awsConfig, awsSession)
	if err != nil {
		return Config{}, err
	}
	// assume that a check has been done before calling this function that
	// no ecsfargateconfig exists
	config.ExecutionPlatforms = append(config.ExecutionPlatforms, ec2Config)
	return config, nil
}

func initializeEc2(awsConfig AWSConfig, awsSession *session.Session) (ExecutionPlatform, error) {

	// create session
	awsSession, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		fmt.Printf("Error initializing AWS session: %v", err)
		return ExecutionPlatform{}, err
	}

	keyOptions, err := getKeyList(awsSession)
	if err != nil {
		fmt.Printf("Error retrieving key list: %v", err)
		return ExecutionPlatform{}, err
	}

	keyName := ReadOptionInput("Select default Ec2 SSH Key", "", keyOptions)
	ec2Config := Ec2Config{
		DefaultLoginKeyName: keyName,
	}

	return ExecutionPlatform{
		Type:          Ec2,
		Configuration: ec2Config,
	}, nil
}

func initializeEcsFargate(awsConfig AWSConfig, awsSession *session.Session) (ExecutionPlatform, error) {

	ecsFargateConfig := EcsFargateConfig{}
	cluster, err := createEcsCluster(awsSession)
	if err != nil {
		fmt.Printf("Error creating ECS cluster...%v", err.Error())
		os.Exit(1)
	}
	fmt.Printf("Successfully created ECS cluster: %s\n", *cluster.ClusterName)
	ecsFargateConfig.ClusterName = *cluster.ClusterName

	return ExecutionPlatform{
		Type:          EcsFargate,
		Configuration: ecsFargateConfig,
	}, nil
}

func ConfigAddEcsFargateExecutionPlatform(config Config) (Config, error) {
	// first check if config contains a valid AWS config...
	var awsConfig AWSConfig
	var err error

	awsSession, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		fmt.Printf("Error initializing AWS session: %v", err)
		return Config{}, err
	}

	if !config.ProviderConfiguration.AWS.Initialized {
		awsConfig, err = InitializeAWS()
		if err != nil {
			return Config{}, err
		}
		config.ProviderConfiguration.AWS = awsConfig
	}

	// with a valid AWS config, we initialize ECS Fargate
	ecsFargateConfig, err := initializeEcsFargate(awsConfig, awsSession)
	if err != nil {
		return Config{}, err
	}
	// assume that a check has been done before calling this function that
	// no ecsfargateconfig exists
	config.ExecutionPlatforms = append(config.ExecutionPlatforms, ecsFargateConfig)
	return config, nil
}
