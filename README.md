![alt tag](https://travis-ci.org/Hyperpilotio/daemon-ecs-scheduler.svg?branch=master)

# daemon-ecs-scheduler
A ECS scheduler that places daemons on every node in ECS cluster

## Quick start

`glide install`

`go build`

`daemon-ecs-scheduler --cluster weave-ecs-demo-cluster --tasks cadvisor:2`

## Feature

* Support the automatic registeration of tasks.

## Help

`daemon-ecs-scheduler  --help`
