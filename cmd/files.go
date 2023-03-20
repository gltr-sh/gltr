package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	gltr "github.com/gltr-sh/gltr/pkg"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v3"
)

func readGltrConfig(configDir string) (c gltr.Config, err error) {
	configFilePath := filepath.Join(configDir, "config.yaml")
	dat, err := os.ReadFile(configFilePath)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(dat, &c)

	// the unmarshalling does not handle array of interfaces well, so we need
	// to do this...
	var typedExecutionPlatformConfigs []gltr.ExecutionPlatform
	for _, e := range c.ExecutionPlatforms {
		switch e.Type {
		case gltr.Docker:
			var pc gltr.DockerConfig
			err := mapstructure.Decode(e.Configuration, &pc)
			if err != nil {
				fmt.Printf("Error decoding docker config: %v\n", err)
				os.Exit(1)
			}
			typedExecutionPlatformConfig := gltr.ExecutionPlatform{
				Type:          gltr.Docker,
				Configuration: pc,
			}
			typedExecutionPlatformConfigs = append(typedExecutionPlatformConfigs, typedExecutionPlatformConfig)
		case gltr.EcsFargate:
			var pc gltr.EcsFargateConfig
			err := mapstructure.Decode(e.Configuration, &pc)
			if err != nil {
				fmt.Printf("Error decoding ECS config: %v\n", err)
				os.Exit(1)
			}
			typedExecutionPlatformConfig := gltr.ExecutionPlatform{
				Type:          gltr.EcsFargate,
				Configuration: pc,
			}
			typedExecutionPlatformConfigs = append(typedExecutionPlatformConfigs, typedExecutionPlatformConfig)
		case gltr.Ec2:
			var pc gltr.Ec2Config
			err := mapstructure.Decode(e.Configuration, &pc)
			if err != nil {
				fmt.Printf("Error decoding Ec2 config: %v\n", err)
				os.Exit(1)
			}
			typedExecutionPlatformConfig := gltr.ExecutionPlatform{
				Type:          gltr.Ec2,
				Configuration: pc,
			}
			typedExecutionPlatformConfigs = append(typedExecutionPlatformConfigs, typedExecutionPlatformConfig)
		}
	}

	c.ExecutionPlatforms = typedExecutionPlatformConfigs
	return
}

// we will allow this to be specified in an env var, but for now, we just
// assume it's ~/.gltr
func getGltrConfigDir() string {
	homeDir := os.Getenv("HOME")
	gltrDir := filepath.Join(homeDir, ".gltr")
	return gltrDir
}

func writeGltrConfig(gltrConfigDir string, c gltr.Config) error {

	err := os.MkdirAll(gltrConfigDir, 0755)
	if err != nil {
		fmt.Printf("Error creating .gltr directory: %v\n", err)
		return err
	}

	configFilePath := filepath.Join(gltrConfigDir, "config.yaml")
	data, err := yaml.Marshal(c)
	err = os.WriteFile(configFilePath, data, 0644)
	if err != nil {
		fmt.Printf("Error writing configuration file: %v\n", err)
		return err
	}
	return nil
}

func readGltrFile(filename string) (gt gltr.Task, err error) {
	dat, err := os.ReadFile(filename)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(dat, &gt)

	// the unmarshalling does not handle array of interfaces well, so we need
	// to do this...
	var typedExecutionPlatformConfigs []gltr.ExecutionPlatformProjectConfig
	for _, e := range gt.ExecutionPlatformConfigs {
		switch e.Type {
		case gltr.Docker:
			var pc gltr.DockerProjectConfig
			err := mapstructure.Decode(e.Configuration, &pc)
			if err != nil {
				fmt.Printf("Error decoding ECS config: %v\n", err)
				os.Exit(1)
			}
			typedExecutionPlatformConfig := gltr.ExecutionPlatformProjectConfig{
				Type:          gltr.Docker,
				Configuration: pc,
			}
			typedExecutionPlatformConfigs = append(typedExecutionPlatformConfigs, typedExecutionPlatformConfig)
		case gltr.EcsFargate:
			var pc gltr.EcsProjectConfig
			err := mapstructure.Decode(e.Configuration, &pc)
			if err != nil {
				fmt.Printf("Error decoding ECS config: %v\n", err)
				os.Exit(1)
			}
			typedExecutionPlatformConfig := gltr.ExecutionPlatformProjectConfig{
				Type:          gltr.EcsFargate,
				Configuration: pc,
			}
			typedExecutionPlatformConfigs = append(typedExecutionPlatformConfigs, typedExecutionPlatformConfig)
		case gltr.Ec2:
			var pc gltr.Ec2ProjectConfig
			err := mapstructure.Decode(e.Configuration, &pc)
			if err != nil {
				fmt.Printf("Error decoding Ec2 config: %v\n", err)
				os.Exit(1)
			}
			typedExecutionPlatformConfig := gltr.ExecutionPlatformProjectConfig{
				Type:          gltr.Ec2,
				Configuration: pc,
			}
			typedExecutionPlatformConfigs = append(typedExecutionPlatformConfigs, typedExecutionPlatformConfig)
		}
	}

	gt.ExecutionPlatformConfigs = typedExecutionPlatformConfigs
	// fmt.Printf("Ecs confg = %v\n", ecsConfigMap)
	return
}

func fileExists(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		return false
	}

	// file exists
	return true
}

func writeGltrFile(filename string, gt gltr.Task) (err error) {
	dat, err := yaml.Marshal(gt)
	// if err == nil or != nil, we let the caller handle it...
	err = os.WriteFile(filename, dat, 0644)
	return
}
