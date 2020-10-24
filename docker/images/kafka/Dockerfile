FROM ubuntu:20.04

ENV TZ=Etc/UTC
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

RUN apt update && \
apt install -y openjdk-8-jre netcat

ADD ${KAFKA_TGZ} /opt
RUN mv /opt/${KAFKA_FILE} /opt/kafka
COPY start.sh /start.sh

ENTRYPOINT ["/bin/bash", "start.sh"]
