PROMETHEUS_VER=2.30.0
PROMETHEUS_FILE=prometheus-${PROMETHEUS_VER}.linux-amd64

sudo useradd prometheus -m
sudo usermod --shell /bin/bash prometheus
echo -e "prometheus:prometheus" | sudo chpasswd
sudo usermod -aG sudo prometheus

cd ~
wget https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VER}/${PROMETHEUS_FILE}.tar.gz
tar xvzf ${PROMETHEUS_FILE}.tar.gz
sudo mv ~/${PROMETHEUS_FILE} /opt/prometheus
sudo chown prometheus:prometheus /opt/prometheus
rm -rf ${PROMETHEUS_FILE}.tar.gz

cd ~
sudo apt install -y apt-transport-https
sudo apt install -y software-properties-common wget
wget -q -O - https://packages.grafana.com/gpg.key | sudo apt-key add -
echo "deb https://packages.grafana.com/oss/deb stable main" | sudo tee -a /etc/apt/sources.list.d/grafana.list
sudo apt update
sudo apt install -y grafana
