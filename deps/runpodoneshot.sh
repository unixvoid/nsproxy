#!/bin/bash

sudo docker run \
	-d \
	-p 4410:4410 \
	-e APP_PORT=4410 \
	-e CLUSTER_NAME=testapp \
	--name=testapp0 \
	mfaltys/pod:4 demoapp $1
