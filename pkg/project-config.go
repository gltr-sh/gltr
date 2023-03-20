package gltr

import (
	"fmt"
	"strconv"

	"github.com/erikgeiser/promptkit/confirmation"
)

func ProjectAddDockerExecutionPlatform(config Config) (ExecutionPlatformProjectConfig, error) {
	fmt.Printf("WARNING: add docker execution platform not supported yet\n")
	return ExecutionPlatformProjectConfig{}, nil
}

func ProjectAddEc2ExecutionPlatform(gt Task, config Config) (ExecutionPlatformProjectConfig, error) {
	ec2Config := config.GetExecutionPlatformConfig(Ec2).(Ec2Config)

	gpuRequired := ReadConfirmationInput("GPU Required", confirmation.No)

	var instanceTypes []string
	switch gpuRequired {
	case true:
		instanceTypes = []string{"g4dn.xlarge", "g4dn.2xlarge", "g4dn.4xlare"}
	case false:
		instanceTypes = []string{"t2.micro", "t2.small", "t2.medium", "t2.large"}
	}
	instanceType := ReadOptionInput(
		"Default Instance Type", "", instanceTypes)

	// create security group
	// assume 2222 is not in the port list - we need to add a check here - FIXME
	ports := append(gt.Ports, 2222)
	securityGroupName := fmt.Sprintf("%v-ec2", gt.ProjectName)
	securityGroupId, err := CreateNewSecurityGroup(securityGroupName, config.ProviderConfiguration.AWS.VpcID, ports)
	if err != nil {
		return ExecutionPlatformProjectConfig{}, err
	}

	var defaultAmi string
	if gpuRequired {
		defaultAmi = config.ProviderConfiguration.AWS.DefaultAmiGPUImage
	} else {
		defaultAmi = config.ProviderConfiguration.AWS.DefaultAmiCPUImage
	}

	projectConfig := Ec2ProjectConfig{
		GpuRequired:         gpuRequired,
		DefaultInstanceType: instanceType,
		DefaultImage:        defaultAmi,
		KeyName:             ec2Config.DefaultLoginKeyName,
		SubnetID:            config.ProviderConfiguration.AWS.SubnetID,
		SecurityGroupID:     securityGroupId,
	}
	return ExecutionPlatformProjectConfig{
		Type:          Ec2,
		Configuration: projectConfig,
	}, nil
}

func ProjectAddEcsFargateExecutionPlatform(gt Task, config Config) (ExecutionPlatformProjectConfig, error) {
	fmt.Printf("WARNING: add default ECS Fargate execution platform\n")

	ecsConfig := config.GetExecutionPlatformConfig(EcsFargate).(EcsFargateConfig)
	ClusterName := ReadTextInput(
		"Enter Cluster Name",
		ecsConfig.ClusterName,
		ecsConfig.ClusterName,
	)

	var cpuOptions []string
	for i := range FargateOptionsByCpu {
		cpuOptions = append(cpuOptions, fmt.Sprintf("%d", i))
	}
	cpuRequirementsString := ReadOptionInput(
		"Enter CPU Requirements (milliCPUs)", "1024", cpuOptions)
	cpuRequirementsInt, _ := strconv.Atoi(cpuRequirementsString)

	var memoryOptions []string
	for _, m := range FargateOptionsByCpu[cpuRequirementsInt] {
		memoryOptions = append(memoryOptions, fmt.Sprintf("%d", m))
	}
	memoryRequirementsString := ReadOptionInput(
		"Enter Memory Requirements (MB)", "2048", memoryOptions)
	memoryRequirementsInt, _ := strconv.Atoi(memoryRequirementsString)

	// create security group
	securityGroupName := fmt.Sprintf("%v-ecs-fargate", gt.ProjectName)
	securityGroupId, err := CreateNewSecurityGroup(securityGroupName, config.ProviderConfiguration.AWS.VpcID, gt.Ports)
	if err != nil {
		return ExecutionPlatformProjectConfig{}, err
	}

	defaultConfig := EcsProjectConfig{
		CPURequirements:    cpuRequirementsInt,
		MemoryRequirements: memoryRequirementsInt,
		ClusterName:        ClusterName,
		SubnetID:           config.ProviderConfiguration.AWS.SubnetID,
		SecurityGroupID:    securityGroupId,
	}
	return ExecutionPlatformProjectConfig{
		Type:          EcsFargate,
		Configuration: defaultConfig,
	}, nil
}
