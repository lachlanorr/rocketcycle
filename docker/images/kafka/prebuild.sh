set -e

SCALA_VER=2.13
KAFKA_VER=2.6.0
export KAFKA_FILE=kafka_${SCALA_VER}-${KAFKA_VER}
export KAFKA_TGZ=${KAFKA_FILE}.tgz

echo "https://mirrors.sonic.net/apache/kafka/${KAFKA_VER}/${KAFKA_TGZ}"
wget https://mirrors.sonic.net/apache/kafka/${KAFKA_VER}/${KAFKA_TGZ}
