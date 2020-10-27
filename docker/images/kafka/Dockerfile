FROM ubuntu:20.04

ENV TZ=Etc/UTC
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

RUN apt update && \
apt install -y openjdk-8-jre netcat

ADD ${KAFKA_TGZ} /opt
RUN mv /opt/${KAFKA_FILE} /opt/kafka
COPY start.sh /start.sh

ENV JMX_AGENT_JAR=/opt/jmx_exporter/${JMX_AGENT_JAR} \
    JMX_AGENT_CONFIG_BROKER=/opt/jmx_exporter/jmx_agent_kafka.yaml \
    JMX_AGENT_CONFIG_ZOOKEEPER=/opt/jmx_exporter/jmx_agent_zookeeper.yaml
COPY jmx_* /opt/jmx_exporter/

ENTRYPOINT ["/bin/bash", "start.sh"]
