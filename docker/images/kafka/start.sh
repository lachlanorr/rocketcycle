#!/bin/bash

cd /opt/kafka

if [ "$1" = "zookeeper" ] ; then
    bin/zookeeper-server-start.sh config/zookeeper.properties
elif [ "$1" = "broker" ] ; then
    bin/kafka-server-start.sh config/server.properties
else
    echo "usage: start.sh [zookeeper|broker]"
fi
