package gltr

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func getEcsClient() (awsSession *session.Session, ecsClient *ecs.ECS, err error) {
	// initialize AWS session
	awsSession, err = session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})

	if err != nil {
		fmt.Printf("Error initializing AWS session: %v\n", err)
		return nil, nil, err
	}

	ecsClient = ecs.New(awsSession)
	return

}

func getEc2Client() (awsSession *session.Session, ec2Client *ec2.EC2, err error) {
	// initialize AWS session
	awsSession, err = session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})

	if err != nil {
		log.Printf("Error initializing AWS session: %v", err)
		return nil, nil, err
	}

	ec2Client = ec2.New(awsSession)
	return

}

func getCluster(ecsClient *ecs.ECS, clusterName string) (cluster *ecs.Cluster, err error) {

	i := ecs.DescribeClustersInput{
		Clusters: []*string{&clusterName},
	}
	clusters, err := ecsClient.DescribeClusters(&i)
	if err != nil {
		log.Printf("Error listing clusters: %v", err)
		os.Exit(1)
	}
	if len(clusters.Clusters) != 1 {
		log.Printf("Error - cluster not found...")
		os.Exit(1)
	}

	cluster = clusters.Clusters[0]
	return
}
