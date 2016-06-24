#!/bin/bash

sudo docker run \
	-d \
	-p 4410:4410 \
	-e APP_PORT=4410 \
	-e CLUSTER_NAME=testapp \
	--name=testapp0 \
	mfaltys/pod:4 demoapp 192.168.1.9

sudo docker run \
	-d \
	-p 4411:4411 \
	-e APP_PORT=4411 \
	-e CLUSTER_NAME=testapp \
	--name=testapp1 \
	mfaltys/pod:4 demoapp 192.168.1.9

sudo docker run \
	-d \
	-p 4412:4412 \
	-e APP_PORT=4412 \
	-e CLUSTER_NAME=testapp \
	--name=testapp2 \
	mfaltys/pod:4 demoapp 192.168.1.9
