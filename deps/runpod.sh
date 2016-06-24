#!/bin/bash

sudo docker run \
	-d \
	-p 443:443 \
	-e PORT=443 \
	-e CLUSTER_NAME=testapp \
	--name=testapp \
	mfaltys/pod:3 demoapp 192.168.1.9
sudo docker logs -f testapp
