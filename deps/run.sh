#!/bin/sh
VER_NO="v0.0.2b PRE-RELEASE:$DIFF"

echo "daemonize yes" > /redis.conf
echo "dbfilename dump.rdb" >> /redis.conf
echo "dir /redisbackup/" >> /redis.conf
echo "save 30 1" >> /redis.conf

redis-server /redis.conf

echo -e "\e[36m ___ ___ ___ ___ ___ _ _ _ _ \e[39m"
echo -e "\e[36m|   |_ -| . |  _| . |_'_| | |\e[39m"
echo -e "\e[36m|_|_|___|  _|_| |___|_._|_  |\e[39m"
echo -e "\e[36m        |_|             |___|\e[39m"
echo -e "         \e[3m:: \e[31m$VER_NO  \e[39m::\e[0m  "


/nsproxy $@
