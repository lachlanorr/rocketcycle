# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://mozilla.org/MPL/2.0/.

set -e

SCALA_VER=2.13
KAFKA_VER=2.6.1
export KAFKA_FILE=kafka_${SCALA_VER}-${KAFKA_VER}
export KAFKA_TGZ=${KAFKA_FILE}.tgz

wget https://mirrors.sonic.net/apache/kafka/${KAFKA_VER}/${KAFKA_TGZ}

JMX_AGENT_VER=0.14.0
export JMX_AGENT_JAR=jmx_prometheus_javaagent-${JMX_AGENT_VER}.jar
wget https://repo1.maven.org/maven2/io/prometheus/jmx/jmx_prometheus_javaagent/${JMX_AGENT_VER}/${JMX_AGENT_JAR}
