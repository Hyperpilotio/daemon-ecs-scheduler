package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"

	ecs_state "github.com/Wen777/ecs_state"
	cli "github.com/urfave/cli"
)

// StartTask on AWS ECS Cluster
// cluster the name of cluster
// taskArn the ARN of task or the name of task
func StartTask(cluster string, taskArn string) error {

	// List cluster info
	sess, _ := session.NewSession(&aws.Config{Region: aws.String("us-west-1")})
	client := ecs.New(sess)

	state := ecs_state.Initialize(cluster, client)
	state.RefreshClusterState()
	state.RefreshContainerInstanceState()
	state.RefreshTaskState()

	clusterInstance := state.FindClusterByName(cluster)

	if clusterInstance.ARN == "" {
		return errors.New("cluster doesn't exist")
	}

	if clusterInstance.Status != "ACTIVE" {
		return errors.New("the cluster is not active")
	}

	instancesList := clusterInstance.ContainerInstances
	fmt.Printf("Found %+v container instances in the cluster: \n\n", len(instancesList))

	// TODO Register task definitions

	// chosenInstances stores those instances who doesn't have the specific task
	var chosenInstances []*string
	for _, v := range instancesList {
		taskExists := false
		for _, task := range v.Tasks {
			if strings.Contains(task.TaskDefinitionARN, taskArn) {
				taskExists = true
				break
			}
		}
		if !taskExists {
			chosenInstances = append(chosenInstances, aws.String(v.ARN))
		}
	}

	if len(chosenInstances) == 0 {
		return nil
	}

	fmt.Printf("How many instances lack the specific task %+v\n\n", len(chosenInstances))

	params := &ecs.StartTaskInput{
		ContainerInstances: chosenInstances,
		TaskDefinition:     aws.String(taskArn),
		Cluster:            aws.String(cluster),
		StartedBy:          aws.String("hyperpilot-pen"),
	}

	resp, err := client.StartTask(params)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	fmt.Println(resp)
	// TODO Add logger
	// TODO Make it running as linux daemon
	return nil
}
func main() {

	// Parse parameters from command line inpu
	var CLUSTER string
	var TaskDef []string
	app := cli.NewApp()
	app.Name = "daemon-ecs-scheduler"
	app.Usage = "A customized scheduler by hyperpilot for AWS ECS"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "cluster",
			Usage:       "The name of target cluster",
			Destination: &CLUSTER,
		},
		cli.StringSliceFlag{
			Name:  "task-definitions, tasks",
			Usage: "Start which task. Format -tasks task_name -tasks another_task_name",
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.String("cluster") != "" {
			TaskDef = c.StringSlice("task-definitions")
			err := StartTask(CLUSTER, TaskDef[0])
			if err != nil {
				return err
			}
		}
		return nil
	}
	app.Run(os.Args)

}
