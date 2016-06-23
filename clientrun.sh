#!/bin/sh

# this is designed to be a generic runscript that is the be the command
# ran from all docker clients.  this script will take care of the
# registration and then launch whatever app the container is for.

echo "$1 nsproxy_master" >> /etc/hosts
SERVER_IP=nsproxy_master
SERVER_PORT=8080
APP_NAME=$(cat /etc/hostname)
CLUSTER_NAME=$2

curl -d hostname=$APP_NAME -d cluster=$CLUSTER_NAME $SERVER_IP:$SERVER_PORT

echo "app is now running..."

while :
do
	sleep 100
done
