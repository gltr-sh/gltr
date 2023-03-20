package gltr

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/pterm/pterm"
	"github.com/tidwall/gjson"
)

// DockerExecutionPlatform does not contain any state right now
type DockerExecutionPlatform struct {
}

// given a container this function determines if it has the given tag
func (d DockerExecutionPlatform) GetTag(c types.Container, tag string) (value *string) {
	for t, v := range c.Labels {
		if t == tag {
			return &v
		}
	}
	return nil
}

// ListTasks iterates over all containers running on the docker engine and
// only returns those that have gltr tags
func (d DockerExecutionPlatform) ListTasks() ([]types.Container, error) {

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		errorString := fmt.Sprintf("Error creating docker client: %s", err)
		return nil, errors.New(errorString)
	}

	listOptions := types.ContainerListOptions{}
	tasks, err := cli.ContainerList(context.TODO(), listOptions)
	if err != nil {
		errorString := fmt.Sprintf("Error creating docker client: %s", err)
		return nil, errors.New(errorString)
	}

	returnSet := []types.Container{}
	for _, t := range tasks {
		if d.GetTag(t, "gltr-managed") != nil {
			returnSet = append(returnSet, t)
		}
	}

	return returnSet, nil
}

func (d DockerExecutionPlatform) getContainerID(cli *client.Client, taskID string) (containerID string, err error) {

	listOptions := types.ContainerListOptions{}
	tasks, err := cli.ContainerList(context.TODO(), listOptions)
	if err != nil {
		errorString := fmt.Sprintf("Error listing containers: %s", err)
		return "", errors.New(errorString)
	}

	found := false
	for _, t := range tasks {
		gltrTaskID := d.GetTag(t, "gltr-task-id")
		if gltrTaskID == nil {
			continue
		}
		if *gltrTaskID == taskID {
			found = true
			containerID = t.ID
			break
		}
	}
	if !found {
		return "", nil
	}
	return
}

// KillTask kills a task with the given taskID
func (d DockerExecutionPlatform) KillTask(taskID string) error {

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		errorString := fmt.Sprintf("Error creating docker client: %s", err)
		return errors.New(errorString)
	}

	// get cotnainerID from gltr-task-id
	containerID, err := d.getContainerID(cli, taskID)

	// we assume that the container has been created with --rm - this means
	// that we just need to stop it and the removal will automatically happen
	err = cli.ContainerStop(context.TODO(), containerID, container.StopOptions{})
	if err != nil {
		errorString := fmt.Sprintf("Error stopping task with id %s: %v", taskID, err)
		return errors.New(errorString)
	}

	// wait for this container removal to terminate
	terminated := false
	for !terminated {
		_, err = cli.ContainerInspect(context.TODO(), containerID)
		if err != nil {
			// assume error means that the container no longer exists; should to
			// better validation here.
			terminated = true
			break
		}
		time.Sleep(time.Second)
	}
	return nil
}

// RunTask runs a docker container  kills a task with the given taskID
func (d DockerExecutionPlatform) RunTask(gt Task, config Config, gltrPrivateKey []byte, hostname string) error {

	taskID := generateTaskID()
	command := createDockerRunInstruction(gt, config, gltrPrivateKey, taskID, true, hostname, false)
	// fmt.Printf("command: %v\n", command)

	cmd := exec.Command(command[0], command[1:]...)
	if err := cmd.Start(); err != nil {
		pterm.Error.Printf("Error launching container run command %v\n", gt.ProjectName)
		return err
	}
	pterm.Info.Printf("Starting container run command  %v\n", gt.ProjectName)

	// apparently, it is not clear how well this works on non linux systems...
	if err := cmd.Wait(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			errString := fmt.Sprintf("docker command terminated with exit code: %d", exiterr.ExitCode())
			return errors.New(errString)
		}
		// error waiting for command to finish...
		return err
	}
	return nil
}

func (d DockerExecutionPlatform) GetContainerAddressAndPort(
	containerName string,
) (addr string, portBindings []PortBinding) {
	// get the address of the container...

	command := []string{"docker", "inspect", containerName}

	out, err := exec.Command(command[0], command[1:]...).Output()
	if err != nil {
		fmt.Printf("Error obtaining IP address for container: %v\n ", err)
		log.Fatal(err)
	}
	// fmt.Printf("docker inspect output: %v", string(out))
	// this may not be so robust - should be revisited
	ipAddressJSON := gjson.Get(string(out), "0.NetworkSettings.Networks.bridge.IPAddress")
	addr = ipAddressJSON.Str

	// this is too selective - we need to make better abstractions here...
	sshPortJSON := gjson.Get(string(out), "0.NetworkSettings.Ports.22/tcp.0.HostPort")
	sshPortString := sshPortJSON.Str

	sshPort, err := strconv.Atoi(sshPortString)
	if err != nil {
		pterm.Error.Printf("Error obtaining host port information\n")
		return
	}

	portBindings = []PortBinding{
		{ContainerPort: 22, HostPort: int(sshPort)},
	}
	return
}
