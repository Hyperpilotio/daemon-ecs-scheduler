package main

import (
	"errors"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"

	"github.com/golang/glog"

	ecs_state "github.com/Wen777/ecs_state"
	cli "github.com/urfave/cli"
)

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
// cluster the name of cluster
// taskArn the ARN of task or the name of task
func StartTask(cluster string, taskDefinitions []string, awsRegion string) error {
	// List cluster info
	sess, _ := session.NewSession(&aws.Config{Region: aws.String(awsRegion)})
	client := ecs.New(sess)

	state := ecs_state.Initialize(cluster, client)
	state.RefreshClusterState()
	state.RefreshContainerInstanceState()
	state.RefreshTaskState()

	clusterInstance := state.FindClusterByName(cluster)

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

	// TODO Register task definitions

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
			Cluster:            aws.String(cluster),
			StartedBy:          aws.String("daemon-ecs-scheduler"),
		}

		resp, err := client.StartTask(params)
		if err != nil {
			glog.Error(err.Error())
			return err
		}

		glog.V(2).Infoln(resp)

	}
	return nil
}

// StartServer start a web server
func StartServer(port string, mode string) error {
	gin.SetMode(mode)

	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	return router.Run(":" + port)
}

func main() {

	// Parse parameters from command line inpu
	var cluster string
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
			Destination: &cluster,
		},
		cli.StringSliceFlag{
			Name:  "task-definitions, tasks",
			Usage: "Start which task. Format -tasks task_name -tasks another_task_name",
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.String("cluster") != "" {
			taskDefinitions = c.StringSlice("task-definitions")
			err := StartTask(cluster, taskDefinitions, awsRegion)
			if err != nil {
				return cli.NewExitError(err.Error(), 1)
			}
		}
		return nil
	}

	// Add sub-command
	// Two parameters, the default value of port is 7777 and the default value of mode is `release`
	app.Commands = []cli.Command{
		{
			Name:  "server",
			Usage: "start a HTTP server. Default value is 8080.",

			Action: func(c *cli.Context) (err error) {

				err = StartServer(c.String("port"), c.String("mode"))
				if err != nil {
					return cli.NewExitError(err.Error(), 1)
				}
				return
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
