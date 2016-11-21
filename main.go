package main

import (
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
func StartTask(CLUSTER string, TASK string) error {

	// List cluster info
	sess, _ := session.NewSession(&aws.Config{Region: aws.String("us-west-1")})
	client := ecs.New(sess)

	state := ecs_state.Initialize(CLUSTER, client)
	state.RefreshClusterState()
	state.RefreshContainerInstanceState()
	state.RefreshTaskState()
	fmt.Printf("Found Cluster: %+v\n\n", len(state.FindClusterByName(CLUSTER).ContainerInstances))

	// TODO Register task definitions
	ec2Arr := state.FindClusterByName(CLUSTER).ContainerInstances

	var arrOfMissing []*string
	for _, v := range ec2Arr {
		flag := 0
		for _, task := range v.Tasks {
			if strings.Contains(task.TaskDefinitionARN, TASK) {
				flag = 1
				break
			}
		}
		if flag == 0 {
			arrOfMissing = append(arrOfMissing, aws.String(v.ARN))
		}
	}

	fmt.Printf("How many containers lack the specific task %+v\n\n", len(arrOfMissing))

	params := &ecs.StartTaskInput{
		ContainerInstances: arrOfMissing,
		TaskDefinition:     aws.String(TASK),
		Cluster:            aws.String(CLUSTER),
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
	app.Name = "hyperpen"
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
