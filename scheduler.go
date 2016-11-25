package main

import (
	"errors"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"

	"github.com/golang/glog"

	ecs_state "github.com/Wen777/ecs_state"
	cli "github.com/urfave/cli"
)

var (
	// Sess session of aws
	Sess *session.Session

	// Client of aws
	Client *ecs.ECS

	// State instance of ecs_state
	State ecs_state.StateOps

	// Cluster the name of cluster
	Cluster string
)

func initialize(cluster string, awsRegion string) {
	Sess, _ = session.NewSession(&aws.Config{Region: aws.String(awsRegion)})
	Client = ecs.New(Sess)
	State = ecs_state.Initialize(cluster, Client)
	State.RefreshClusterState()
	State.RefreshContainerInstanceState()
	State.RefreshTaskState()
}

// selectUnlaunchedInstances finds instances that has launched the specified task
// instances all the instances in the cluster
// newTask the task name of the new task
func selectUnlaunchedInstances(instances []ecs_state.ContainerInstance, newTask string) []*string {
	var chosenInstances []*string
	for _, v := range instances {
		taskExists := false
		for _, runningTask := range v.Tasks {
			if strings.Contains(runningTask.TaskDefinitionARN, newTask) {
				taskExists = true
				break
			}
		}
		if !taskExists {
			chosenInstances = append(chosenInstances, aws.String(v.ARN))
		}
	}
	return chosenInstances
}

// StartTask on AWS ECS Cluster
// taskArn the ARN of task or the name of task
func StartTask(taskDefinitions []string) error {
	// Cluster comes from global variable which is determined on the start of runtime of scheduler.
	clusterInstance := State.FindClusterByName(Cluster)

	if clusterInstance.ARN == "" {
		glog.Error("cluster doesn't exist")
		return errors.New("cluster doesn't exist")
	}

	if clusterInstance.Status != "ACTIVE" {
		glog.Error("the cluster is not active")
		return errors.New("the cluster is not active")
	}

	instances := clusterInstance.ContainerInstances
	glog.V(3).Infoln("Found %+v container instances in the cluster: ", len(instances))

	// chosenInstances stores those instances who doesn't have the specific task
	for _, task := range taskDefinitions {
		chosenInstances := selectUnlaunchedInstances(instances, task)
		if len(chosenInstances) == 0 {
			return nil
		}

		glog.V(3).Infoln("How many instances lack the specific task %+v\n\n", len(chosenInstances))

		params := &ecs.StartTaskInput{
			ContainerInstances: chosenInstances,
			TaskDefinition:     aws.String(task),
			Cluster:            aws.String(Cluster),
			StartedBy:          aws.String("daemon-ecs-scheduler"),
		}

		resp, err := Client.StartTask(params)

		glog.V(2).Infoln(resp)

		if len(resp.Failures) != 0 {
			glog.Warningf("[StartTask] %v", resp.Failures)
		}

		if err != nil {
			glog.Error(err.Error())
			return err
		}

	}
	return nil
}

func main() {

	// Parse parameters from command line inpu
	var awsRegion string
	var taskDefinitions []string

	app := cli.NewApp()
	app.Name = "daemon-ecs-scheduler"
	app.Usage = "A daemon scheduler by hyperpilot for AWS ECS"
	// Global flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "aws-region",
			Usage:       "The targetted AWS region",
			Destination: &awsRegion,
		},
		cli.StringFlag{
			Name:        "cluster",
			Usage:       "The name of target cluster",
			Destination: &Cluster,
		},
		cli.StringSliceFlag{
			Name:  "task-definitions, tasks",
			Usage: "Start which task. Format -tasks task_name -tasks another_task_name",
		},
	}
	app.Action = func(c *cli.Context) error {
		if Cluster == "" {
			return cli.NewExitError("cluster does exit", 1)
		}
		taskDefinitions = c.StringSlice("task-definitions")
		if len(taskDefinitions) == 0 {
			return cli.NewExitError("no tasks", 1)
		}

		initialize(Cluster, awsRegion)

		err := StartTask(taskDefinitions)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}
		return nil
	}

	// Add sub-command
	// Two parameters, the default value of port is 7777 and the default value of mode is `release`
	app.Commands = []cli.Command{
		{
			Name:  "server",
			Usage: "start a HTTP server. Default value is 8080.",

			Action: func(c *cli.Context) error {
				if Cluster == "" {
					return cli.NewExitError("[server] cluster does exit", 1)
				}
				if awsRegion == "" {
					return cli.NewExitError("[server] awsRegion is undefined", 1)
				}

				initialize(Cluster, awsRegion)

				err := StartServer(c.String("port"), c.String("mode"))
				if err != nil {
					return cli.NewExitError(err.Error(), 1)
				}
				return nil
			}, Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "port",
					Value: "7777",
					Usage: "The port of HTTP server",
				},
				cli.StringFlag{
					Name:  "mode",
					Value: "release",
					Usage: "The mode of HTTP server. \"release\", \"debug\", and \"test\" mode.",
				},
			},
		},
	}
	app.Run(os.Args)
}
