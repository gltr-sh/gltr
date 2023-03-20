package gltr

import (
	"fmt"
	"os"
)

func CreateNewSecurityGroup(securityGroupName, vpcID string, ports []int) (securityGroupID string, err error) {
	_, ec2Client, err := getEc2Client()

	securityGroupID, err = createSecurityGroup(ec2Client, vpcID, securityGroupName)

	for _, p := range ports {
		err = addSecurityGroupRule(ec2Client, securityGroupID, p)
		if err != nil {
			fmt.Printf("Error adding security group rules to security group %v\n", err)
			os.Exit(1)
		}
	}
	return

}
