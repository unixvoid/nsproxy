#!/bin/sh

# this is designed to be a generic runscript that is to be the command
# ran from all docker clients.  this script will take care of the
# registration and then launch whatever app the container is for.

# client must set environment variable 'NSPROXY_MASTER'
NSPROXY_MASTER_IP=$(drill $NSPROXY_MASTER | grep -o '[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}\.[0-9]\{1,3\}' | head -1)

echo "$NSPROXY_MASTER_IP nsproxy_master" >> /etc/hosts
echo "nameserver $NSPROXY_MASTER_IP" > /etc/resolv.conf
SERVER_IP=nsproxy_master
SERVER_PORT=8080
APP_NAME=$(cat /etc/hostname)

curl -d hostname=$APP_NAME -d cluster=$CLUSTER_NAME -d port=$APP_PORT $SERVER_IP:$SERVER_PORT

echo "app is now running..."

while :
do
	sleep 100
done
