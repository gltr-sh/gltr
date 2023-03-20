package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	gltr "github.com/gltr-sh/gltr/pkg"
	"github.com/google/go-github/github"
)

func lookupSSHKeysWithUsername(u string) (keys []string) {

	request := fmt.Sprintf("https://github.com/%s.keys", u)
	resp, err := http.Get(request)
	if err != nil {
		fmt.Printf("Error searching for users\n")
	}

	body, _ := ioutil.ReadAll(resp.Body)
	keys = strings.Split(string(body), "\n")
	fmt.Printf("response = %v\n", string(body))
	return
}

func lookupSshKeysWithEmailAddr(e string) (key string) {
	client := github.NewClient(nil)

	ctx := context.Background()
	// res, resp, err := client.Search.Users(ctx, "", nil)
	query := fmt.Sprintf("q=%s", e)
	fmt.Printf("looking up user with query %v\n", query)
	resp, _, err := client.Search.Users(ctx, query, nil)
	if err != nil {
		fmt.Printf("Error searching for users\n")
	}

	fmt.Printf("response = %v\n", resp.Users)
	return
}

var (
	defaultContainerImage = "gltr/minimal-notebook"
)

func extractProjectShortNameFromRepo(repoName string) string {
	splitArray := strings.Split(repoName, "/")
	lastString := splitArray[len(splitArray)-1]
	splitArray = strings.Split(lastString, ".")
	return splitArray[0]
}

func initializeProjectWithDefaults(defaults gltr.Task, config gltr.Config) (project gltr.Task) {

	// must change this to use promptkit
	project.ProjectName = gltr.ReadTextInput(
		"Enter Project Name",
		defaults.ProjectName,
		"Project name cannot be empty",
	)
	project.ContainerImage = gltr.ReadTextInput(
		"Enter Container Image",
		defaults.ContainerImage,
		"Container image cannot be empty",
	)
	if len(config.ExecutionPlatforms) == 1 {
		// we only have one option here; we assume it works...
		project.ExecutionPlatformConfigs = []gltr.ExecutionPlatformProjectConfig{
			{
				Type: gltr.Docker,
				Configuration: gltr.DockerProjectConfig{
					GpuEnabled: false,
				},
			},
		}
	} else {
		var executionPlatforms []string
		for _, p := range config.ExecutionPlatforms {
			executionPlatforms = append(executionPlatforms, p.Type.ToString())
		}
	}

	project.GitRepo = gltr.ReadTextInput("Enter Git Repo (public via https)", defaults.GitRepo,
		"Git repo cannot be empty")

	project.Users = defaults.Users

	openPorts := gltr.ReadTextInput(
		"Enter Open Ports required for this project: ",
		"8888,22",
		"Specified ports cannot be empty",
	)
	portsStringArray := strings.Split(openPorts, ",")
	ports := []int{}
	for _, p := range portsStringArray {
		port, _ := strconv.Atoi(p)
		ports = append(ports, port)
	}
	project.Ports = ports
	project.ProjectID = defaults.ProjectID

	return
}
