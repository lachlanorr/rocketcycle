#!/bin/bash

cd /opt/kafka

if [ "$1" = "zookeeper" ] ; then
    export EXTRA_ARGS="-Dcom.sun.management.jmxremote \
-Dcom.sun.management.jmxremote.authenticate=false \
-Dcom.sun.management.jmxremote.ssl=false \
-Djava.util.logging.config.file=logging.properties \
-javaagent:${JMX_AGENT_JAR}=9870:${JMX_AGENT_CONFIG_ZOOKEEPER}"
    bin/zookeeper-server-start.sh config/zookeeper.properties
elif [ "$1" = "broker" ] ; then
    export EXTRA_ARGS="-Dcom.sun.management.jmxremote \
-Dcom.sun.management.jmxremote.authenticate=false \
-Dcom.sun.management.jmxremote.ssl=false \
-Djava.util.logging.config.file=logging.properties \
-javaagent:${JMX_AGENT_JAR}=9871:${JMX_AGENT_CONFIG_BROKER}"
    bin/kafka-server-start.sh config/server.properties
else
    echo "usage: start.sh [zookeeper|broker]"
fi
