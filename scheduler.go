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
	job "github.com/carlescere/scheduler"
	cli "github.com/urfave/cli"
)

var (
	Sess      *session.Session   // Sess session of aws
	Client    *ecs.ECS           // Client ECS Client
	State     ecs_state.StateOps // State instance of ecs_state
	Cluster   string             // Cluster the name of the ECS cluster
	AWSRegion string             // AWSRegion AWS Region of the cluster
	Interval  int                // Interval the time interval of ecs state refresh
)

// Run starts the scheduler
func Run(port string, isDebug bool) error {
	Sess, _ = session.NewSession(&aws.Config{Region: aws.String(AWSRegion)})
	Client = ecs.New(Sess)
	State = ecs_state.Initialize(Cluster, Client)
	job.Every(Interval).Minutes().Run(Refresh)
	var mode = "release"
	if isDebug {
		mode = "debug"
	}
	return StartServer(port, mode)
}

// Refresh the cluster state
func Refresh() {
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
	// Parse parameters from command line input.
	app := cli.NewApp()
	app.Name = "daemon-ecs-scheduler"
	app.Usage = "A daemon scheduler by hyperpilot for AWS ECS"
	// Global flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "aws-region",
			Usage:       "The targetted AWS region",
			Destination: &AWSRegion,
		},
		cli.StringFlag{
			Name:        "cluster",
			Usage:       "The name of target cluster",
			Destination: &Cluster,
		},
		cli.StringFlag{
			Name:  "port",
			Value: "7777",
			Usage: "The port of scheduler REST server",
		},
		cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug mode",
		},
		cli.IntFlag{
			Name:        "interval",
			Value:       5,
			Usage:       "According to the given time interval (minutes) to refresh the cluster state.",
			Destination: &Interval,
		},
	}
	app.Action = func(c *cli.Context) error {
		if Cluster == "" {
			return cli.NewExitError("Cluster name is required", 1)
		}
		if AWSRegion == "" {
			return cli.NewExitError("AWS Region is required", 1)
		}
		return Run(c.String("port"), c.Bool("mode"))
	}

	app.Run(os.Args)
}
