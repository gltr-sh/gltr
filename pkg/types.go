package gltr

import (
	"errors"
	"time"

	"gopkg.in/yaml.v3"
)

type PortBinding struct {
	ContainerPort int
	HostPort      int
}

type ExecutionPlatformType int

const (
	Docker ExecutionPlatformType = iota
	EcsFargate
	Ec2
	GcpComputeEngine
	UnknownPlatform
)

var ExecutionPlatformMap = map[ExecutionPlatformType]string{
	Docker:           "docker",
	EcsFargate:       "ecs-fargate",
	Ec2:              "ec2",
	GcpComputeEngine: "gcp-compute-engine",
	UnknownPlatform:  "unknown-platform",
}

func ParseExecutionPlatformType(s string) (ExecutionPlatformType, error) {
	found := false
	var returnVal ExecutionPlatformType
	for t, tString := range ExecutionPlatformMap {
		if tString == s {
			found = true
			returnVal = t
		}
	}
	if found == false {
		return UnknownPlatform, errors.New("unknown execution platform")
	}
	return returnVal, nil
}

func (e ExecutionPlatformType) ToString() string {
	platformTypeString, ok := ExecutionPlatformMap[e]
	if ok {
		return platformTypeString
	}

	return "UnknownPlatform"
}

type ExecutionPlatformConfiguration interface{}

type ExecutionPlatform struct {
	Type          ExecutionPlatformType
	Configuration ExecutionPlatformConfiguration
}

// this is what an ExecutionPlatform should provide - at least
// initially - might merge this with the ExecutionPlatform struct above
type ExecutionPlatformInterface interface {
	RunTask(task Task) (taskID string, err error)
	ListTasks() (runningTasks string, err error)
	KillTask(taskId string) error
}

func (e *ExecutionPlatformType) UnmarshalYAML(n *yaml.Node) error {
	// unmarshal func(interface{}) error,
	platformTypeString := string(n.Value)

	platform, err := ParseExecutionPlatformType(platformTypeString)
	if err != nil {
		return err
	}
	*e = platform
	return nil
}

func (e ExecutionPlatformType) MarshalYAML() (interface{}, error) {
	return ExecutionPlatformMap[e], nil
}

// the following is v basic; it;s is a placefolder for now
type Task struct {
	ProjectID                string                           `json:"project_id"                 yaml:"project_id"`
	ProjectName              string                           `json:"project_name"               yaml:"project_name"`
	ContainerImage           string                           `json:"container_image"            yaml:"container_image"`
	GpuRequired              bool                             `json:"gpu_required"               yaml:"gpu_required"`
	DefaultExecutionPlatform ExecutionPlatformType            `json:"default_execution_platform" yaml:"default_execution_platform"`
	GitRepo                  string                           `json:"git_repo"                   yaml:"git_repo"`
	ProjectPublicKey         string                           `json:"project_public_key"         yaml:"project_public_key"`
	Users                    []User                           `json:"users"                      yaml:"users"`
	ExecutionPlatformConfigs []ExecutionPlatformProjectConfig `json:"execution_platform_configs" yaml:"execution_platform_configs"`
	Ports                    []int                            `json:"ports"                      yaml:"ports"`
}

func (d TaskEc2Config) Type() ExecutionPlatformType {
	return Ec2
}

type TaskEc2Config struct {
	AmiImage     string `json:"ami_image"     yaml:"ami_image"`
	InstanceType string `json:"instance_type" yaml:"instance_type"`
	KeyName      string `json:"key_name"      yaml:"key_name"`
}

type Ec2Config struct {
	// could be able to add some things here about volumes and EFS but ignore for now
	DefaultLoginKeyName string `json:"default_login_key_name" yaml:"default_login_key_name" mapstructure:"default_login_key_name"`
}

type EcsFargateConfig struct {
	CPURequirements    int    `json:"cpu_requirements"    yaml:"cpu_requirements"    mapstructure:"cpu_requirements"`
	MemoryRequirements int    `json:"memory_requirements" yaml:"memory_requirements" mapstructure:"memory_requirements"`
	ClusterName        string `json:"cluster_name"        yaml:"cluster_name"        mapstructure:"cluster_name"`
}

type DockerConfig struct {
	Enabled       bool     `json:"enabled"               yaml:"enabled"               mapstructure:"enabled"`
	EngineVersion string   `json:"docker_engine_version" yaml:"docker_engine_version" mapstructure:"docker_engine_version"`
	Runtimes      []string `json:"docker_runtimes"       yaml:"docker_runtimes"       mapstructure:"docker_runtimes"`
}

type AWSConfig struct {
	Initialized        bool   `json:"initialized"           yaml:"initialized"`
	Enabled            bool   `json:"enabled"               yaml:"enabled"`
	SubnetID           string `json:"subnet_id"             yaml:"subnet_id"`
	RegionName         string `json:"region_name"           yaml:"region_name"`
	VpcID              string `json:"vpc_id"                yaml:"vpc_id"`
	IgwID              string `json:"igw_id"                yaml:"igw_id"`
	DefaultAmiCPUImage string `json:"default_ami_cpu_image" yaml:"default_ami_cpu_image"`
	DefaultAmiGPUImage string `json:"default_ami_gpu_image" yaml:"default_ami_gpu_image"`
}

type AzureConfig struct {
	Enabled     bool   `json:"enabled"      yaml:"enabled"`
	ClusterName string `json:"cluster_name" yaml:"cluster_name"`
}

type GCPConfig struct {
	Enabled         bool   `json:"enabled"          yaml:"enabled"`
	CredentialsFile string `json:"credentials_file" yaml:"credentials_file"`
}

type User struct {
	Name   string `json:"name"    yaml:"name"`
	Email  string `json:"email"   yaml:"email"`
	SshKey string `json:"ssh_key" yaml:"ssh_key"`
}

// this structs contains basic provider configuration information for each of the
// providers; this can span multiple services on a single provider
type ProviderConfiguration struct {
	AWS AWSConfig `json:"aws" yaml:"aws"`
	GCP GCPConfig `json:"gcp" yaml:"gcp"`
}

type Config struct {
	ExecutionPlatforms    []ExecutionPlatform   `json:"execution_platforms"    yaml:"execution_platforms"`
	ProviderConfiguration ProviderConfiguration `json:"provider_configuration" yaml:"provider_configuration"`
	LastUpdate            time.Time             `json:"last_update"            yaml:"last_update"`
	User                  User                  `json:"user"                   yaml:"user"`
}

type ExecutionPlatformProjectConfiguration interface{}

type ExecutionPlatformProjectConfig struct {
	Type          ExecutionPlatformType                 `json:"type"          yaml:"type"`
	Configuration ExecutionPlatformProjectConfiguration `json:"configuration" yaml:"configuration"`
}

type DockerProjectConfig struct {
	GpuEnabled bool `json:"gpu_enabled" yaml:"gpu_enabled" mapstructure:"gpu_enabled"`
}

type EcsProjectConfig struct {
	CPURequirements    int    `json:"cpu_requirements"    yaml:"cpu_requirements"    mapstructure:"cpu_requirements"`
	MemoryRequirements int    `json:"memory_requirements" yaml:"memory_requirements" mapstructure:"memory_requirements"`
	ClusterName        string `json:"cluster_name"        yaml:"cluster_name"        mapstructure:"cluster_name"`
	SubnetID           string `json:"subnet_id"           yaml:"subnet_id"           mapstructure:"subnet_id"`
	SecurityGroupID    string `json:"security_group_id"   yaml:"security_group_id"   mapstructure:"security_group_id"`
}

type Ec2ProjectConfig struct {
	GpuRequired         bool   `json:"gpu_required"          yaml:"gpu_required"          mapstructure:"gpu_required"`
	DefaultInstanceType string `json:"default_instance_type" yaml:"default_instance_type" mapstructure:"default_instance_type"`
	DefaultImage        string `json:"default_image"         yaml:"default_image"         mapstructure:"default_image"`
	KeyName             string `json:"key_name"              yaml:"key_name"              mapstructure:"key_name"`
	SubnetID            string `json:"subnet_id"             yaml:"subnet_id"             mapstructure:"subnet_id"`
	SecurityGroupID     string `json:"security_group_id"     yaml:"security_group_id"     mapstructure:"security_group_id"`
}

func (t Task) GetExecutionPlatformProjectConfig(
	platformType ExecutionPlatformType,
) ExecutionPlatformProjectConfiguration {
	for _, c := range t.ExecutionPlatformConfigs {
		if c.Type == platformType {
			return c.Configuration
		}
	}
	return nil
}

func (c Config) GetExecutionPlatformConfig(
	platformType ExecutionPlatformType,
) ExecutionPlatformConfiguration {
	for _, conf := range c.ExecutionPlatforms {
		if conf.Type == platformType {
			return conf.Configuration
		}
	}
	return nil
}
