KAFKA_VER=3.0.0
SCALA_VER=2.13
FULL_VER=${SCALA_VER}-${KAFKA_VER}
FILE_NAME=kafka_${FULL_VER}
KAFKA_PATH=/opt/${FILE_NAME}

# install java
sudo apt-get -y install default-jre

# prep kafka user, used for zookeeper and the brokers
sudo useradd kafka -m
sudo usermod --shell /bin/bash kafka
echo -e "kafka:kafka" | sudo chpasswd
sudo usermod -aG sudo kafka

# install kafka binaries
cd /opt
sudo wget https://archive.apache.org/dist/kafka/${KAFKA_VER}/${FILE_NAME}.tgz
sudo tar -xzf ${FILE_NAME}.tgz
sudo mv /opt/${FILE_NAME} /opt/kafka
sudo chown -R kafka:kafka /opt/kafka
