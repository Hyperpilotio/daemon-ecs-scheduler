FROM golang:1.6

RUN curl https://glide.sh/get | sh

ADD . /go/src/github.com/hyperpilotio/daemon-ecs-scheduler

WORKDIR /go/src/github.com/hyperpilotio/daemon-ecs-scheduler

RUN glide install && go build

ENTRYPOINT /go/src/github.com/hyperpilotio/daemon-ecs-scheduler/daemon-ecs-scheduler