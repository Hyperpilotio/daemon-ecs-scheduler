# daemon-ecs-scheduler
A ECS scheduler that places daemons on every node in ECS cluster

## Quick start

`glide install`

`go build`

`daemon-ecs-scheduler --cluster weave-ecs-demo-cluster --tasks cadvisor:2`

## Feature

* Support the automatic registeration of tasks.

