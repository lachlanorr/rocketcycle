NODE_EXPORTER_VER=1.3.1
NODE_EXPORTER_FILE=node_exporter-${NODE_EXPORTER_VER}.linux-amd64

sudo apt-get update
sudo apt-get update # needed, not sure why, but updates come down the second time
sudo apt-get upgrade

sudo useradd metrics -m
sudo usermod --shell /bin/bash metrics
echo -e "metrics:metrics" | sudo chpasswd
sudo usermod -aG sudo metrics

cd ~
wget https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VER}/${NODE_EXPORTER_FILE}.tar.gz
tar xvzf ${NODE_EXPORTER_FILE}.tar.gz
sudo mv ~/${NODE_EXPORTER_FILE}/node_exporter /usr/local/bin/node_exporter
sudo chown root:root /usr/local/bin/node_exporter
rm -rf ~/${NODE_EXPORTER_FILE}*
