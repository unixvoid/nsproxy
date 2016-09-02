#!/bin/bash

sudo rkt run \
	--insecure-options=all \
	--net=host \
        --volume redis,kind=host,source=/tmp/ \
	--debug \
        ./nsproxy.aci

#CURRENT_DIR=$(pwd)
#--port=dns:8053 \
#--port=api:8085 \
#--volume redis,kind=host,source=$CURRENT_DIR \
#--net=host \
