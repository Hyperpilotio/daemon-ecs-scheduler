# daemon-ecs-scheduler
A ECS scheduler that places daemons on every node in ECS cluster

## Quick start

`glide install`

`go build`

`daemon-ecs-scheduler --cluster weave-ecs-demo-cluster --tasks cadvisor:2`

***Beware of the mode of gin web server, the default mode is debug mode.***

```
- using env:	export GIN_MODE=release
- using code:	gin.SetMode(gin.ReleaseMode)
```

## Feature

* Support the automatic registeration of tasks.

## Help

`daemon-ecs-scheduler  --help`
