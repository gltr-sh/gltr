/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	gltr "github.com/gltr-sh/gltr/pkg"
	"github.com/kevinburke/ssh_config"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	gltrSSHConfigFile = "config.gltr"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute a gltr task on the configured execution platform",
	Run:   runCommand,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringP("file", "f", "gltr.yaml", "gltr yaml file")
	runCmd.Flags().BoolP("docker", "", false, "Run gltr task on local docker engine")
	runCmd.Flags().BoolP("ecs-fargate", "", false, "Run gltr task on AWS ECS Fargate")
	runCmd.Flags().BoolP("ec2", "", false, "Run gltr task on AWS EC2")
	runCmd.Flags().BoolP("gcp", "", false, "Run gltr task on GCP")
}

func oneIfTrue(b bool) int {
	if b {
		return 1
	}
	return 0
}

func getExecutionPlatform(gt gltr.Task, runDocker, runEcsFargate, runEc2, runGcp bool) gltr.ExecutionPlatformType {

	// first need to figure out where to run this...
	platformsToRun := 0
	platformsToRun += oneIfTrue(runDocker)
	platformsToRun += oneIfTrue(runEcsFargate)
	platformsToRun += oneIfTrue(runEc2)
	platformsToRun += oneIfTrue(runGcp)

	if platformsToRun > 1 {
		fmt.Printf("Cannot specify more than one execution platform...exiting")
		os.Exit(1)
	}

	if platformsToRun == 0 {
		// then we get the default execution platform
		return gt.DefaultExecutionPlatform
	}

	switch {
	case runDocker == true:
		return gltr.Docker
	case runEcsFargate == true:
		return gltr.EcsFargate
	case runEc2 == true:
		return gltr.Ec2
	case runGcp == true:
		return gltr.GcpComputeEngine
	}
	return gltr.UnknownPlatform
}

func writeSSHConfig(filename string, cfg *ssh_config.Config) (err error) {
	err = ioutil.WriteFile(filepath.Join(os.Getenv("HOME"), ".ssh", filename), []byte(cfg.String()), 0644)
	return
}

func readSSHConfig(filename string) (cfg *ssh_config.Config, err error) {

	f, err := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", filename))
	if err != nil {
		return nil, err
	}

	if cfg, err = ssh_config.Decode(f); err != nil {
		return nil, err
	}

	return cfg, nil
}

func findHost(cfg *ssh_config.Config, hostname string) *ssh_config.Host {
	for i, h := range cfg.Hosts {
		// the first element matches everything and provides defaults
		if i != 0 {
			if h.Matches(hostname) {
				return h
			}
		}
	}
	return nil
}

func addHostToSSHConfig(sshHostEntry, host string, port int) error {

	pterm.Info.Printf("Adding host to ssh config\n")

	portString := fmt.Sprintf("%v", port)

	config, err := readSSHConfig(gltrSSHConfigFile)
	if err != nil {
		pterm.Info.Printf("No ssh config file exists - creating...\n")
		newHostPattern, _ := ssh_config.NewPattern(sshHostEntry)
		newHost := ssh_config.Host{
			Patterns: []*ssh_config.Pattern{newHostPattern},
			Nodes: []ssh_config.Node{
				&ssh_config.KV{Key: "hostname", Value: host},
				&ssh_config.KV{Key: "user", Value: "gltr"},
				&ssh_config.KV{Key: "port", Value: portString},
				&ssh_config.KV{Key: "ForwardAgent", Value: "yes"},
				// this adds an empty line after the defined kv pairs as a separator
				&ssh_config.Empty{},
			},
		}
		config := ssh_config.Config{
			Hosts: []*ssh_config.Host{&newHost},
		}
		err = writeSSHConfig(gltrSSHConfigFile, &config)
		if err == nil {
			pterm.Success.Printf("Host configuration written to new ssh config file\n")
		}
		return err
	}

	// check if the host is already defined
	definedHost := findHost(config, sshHostEntry)
	if definedHost != nil {
		// then the record exists, but we want to update it...
		hostFound := false
		portFound := false
		for _, n := range definedHost.Nodes {
			switch n.(type) {
			case *ssh_config.KV:
				nodeKV := n.(*ssh_config.KV)
				if nodeKV.Key == "hostname" {
					nodeKV.Value = host
					hostFound = true
				}
				if nodeKV.Key == "port" {
					nodeKV.Value = portString
					portFound = true
				}
			}
		}
		if !hostFound {
			definedHost.Nodes = append(definedHost.Nodes, &ssh_config.KV{Key: "hostname", Value: host})
		}
		if !portFound {
			definedHost.Nodes = append(definedHost.Nodes, &ssh_config.KV{Key: "port", Value: portString})
		}
		// definedHost.Set(sshHostEntry, "hostname", host)
	} else {
		// thore is no entry for this host, so we need to create one...
		newHostPattern, err := ssh_config.NewPattern(sshHostEntry)
		if err != nil {
			return err
		}
		newHost := ssh_config.Host{
			Patterns: []*ssh_config.Pattern{newHostPattern},
			Nodes: []ssh_config.Node{
				&ssh_config.KV{Key: "hostname", Value: host},
				&ssh_config.KV{Key: "user", Value: "gltr"},
				&ssh_config.KV{Key: "port", Value: portString},
				&ssh_config.KV{Key: "ForwardAgent", Value: "yes"},
				// this adds an empty line after the defined kv pairs as a separator
				&ssh_config.Empty{},
			},
		}
		// add the new host to the existing ssh config...
		config.Hosts = append(config.Hosts, &newHost)
	}

	err = writeSSHConfig(gltrSSHConfigFile, config)
	if err == nil {
		pterm.Success.Printf("Ssh config updated with new host configuration\n")
	}
	return err
}

func runCommand(cmd *cobra.Command, args []string) {
	gltrFilename, _ := cmd.Flags().GetString("file")
	runDocker, _ := cmd.Flags().GetBool("docker")
	runEcsFargate, _ := cmd.Flags().GetBool("ecs-fargate")
	runEc2, _ := cmd.Flags().GetBool("ec2")
	runGcp, _ := cmd.Flags().GetBool("gcp")

	gltrConfigDir := getGltrConfigDir()
	config, err := readGltrConfig(gltrConfigDir)
	if err != nil {
		fmt.Printf("No existing configuration - creating new configuration...\n")
	}

	gt, err := readGltrFile(gltrFilename)
	if err != nil {
		fmt.Printf("Error reading gltr file - exiting: %v", err.Error())
		os.Exit(1)
	}

	executionPlatform := getExecutionPlatform(gt, runDocker, runEcsFargate, runEc2, runGcp)

	privateKey, err := readPrivateKey(getGltrConfigDir(), gt.ProjectID)
	if err != nil {
		fmt.Printf("Error reading private key: %v\n", err)
		os.Exit(1)
	}

	switch executionPlatform {
	case gltr.Docker:
		pterm.Info.Printf("Running task on local docker engine\n")
		dockerExecutionPlatform := gltr.DockerExecutionPlatform{}
		hostname := fmt.Sprintf("%s-docker", gt.ProjectName)
		err := dockerExecutionPlatform.RunTask(gt, config, privateKey, hostname)
		if err != nil {
			pterm.Error.Printf("Error running docker container: %v - exiting...\n", err)
			os.Exit(1)
		}
		if err != nil {
			pterm.Error.Printf("Error launching container: %v\n", err)
			os.Exit(1)
		}
		containerIPAddress, portBindings := dockerExecutionPlatform.GetContainerAddressAndPort(gt.ProjectName)
		pterm.Success.Printf("Container running at %v\n", containerIPAddress)
		err = addHostToSSHConfig(hostname, "localhost", portBindings[0].HostPort)
		if err != nil {
			pterm.Error.Printf("Error adding host to ssh config: %v\n", err)
			os.Exit(1)
		}
		pterm.Info.Printf("Access container using: ssh %v\n", hostname)
	case gltr.Ec2:
		startTime := time.Now()
		pterm.Info.Printf("Running task on Ec2 (start time %v)\n", startTime.Format(time.RFC3339))
		hostname := fmt.Sprintf("%s-ec2", gt.ProjectName)
		host, err := gltr.RunAwsEc2(gt, config, privateKey, hostname)
		if err != nil {
			pterm.Error.Printf("Error launching workspace on Ec2: %v\n", err)
			os.Exit(1)
		}
		err = addHostToSSHConfig(hostname, host, 22)
		if err != nil {
			pterm.Error.Printf("Error adding host to ssh config: %v\n", err)
			os.Exit(1)
		}
		pterm.Info.Printf("Finish time: %v - duration %v\n", time.Now().Format(time.RFC3339), time.Now().Sub(startTime))
		pterm.Info.Printf("Access container using: ssh %v\n", hostname)
	case gltr.EcsFargate:
		pterm.Info.Printf("Running workspace on ECS Fargate\n")
		hostname := fmt.Sprintf("%s-ecs-fargate", gt.ProjectName)
		host, err := gltr.RunAwsEcs(gt, config, privateKey, hostname)
		if err != nil {
			pterm.Error.Printf("Error launching workspace on Ecs Fargate: %v\n", err)
			os.Exit(1)
		}
		err = addHostToSSHConfig(hostname, host, 22)
		if err != nil {
			pterm.Error.Printf("Error adding host to ssh config: %v\n", err)
			os.Exit(1)
		}
		pterm.Info.Printf("Access container using: ssh %v\n", hostname)
	case gltr.GcpComputeEngine:
		fmt.Printf("GCP currently not supported\n")
		os.Exit(1)
	case gltr.UnknownPlatform:
		fmt.Printf("No execution platform defined\n")
		os.Exit(1)
	}
}
