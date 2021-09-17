JAEGER_VER=1.26.0
JAEGER_FILE=jaeger-${JAEGER_VER}-linux-amd64

OTELCOL_VER=0.35.0
OTELCOL_FILE=otelcol_${OTELCOL_VER}_linux_amd64

#GO_VER=1.17
#GO_FILE_NAME=go${GO_VER}.linux-amd64.tar.gz

sudo apt update
sudo apt update # needed, not sure why, but updates come down the second time
sudo apt upgrade

sudo useradd telem -m
sudo usermod --shell /bin/bash telem
echo -e "telem:telem" | sudo chpasswd
sudo usermod -aG sudo telem

cd ~
wget https://github.com/jaegertracing/jaeger/releases/download/v${JAEGER_VER}/${JAEGER_FILE}.tar.gz
tar xvzf ${JAEGER_FILE}.tar.gz
sudo cp ${JAEGER_FILE}/jaeger-* /usr/local/bin
rm -rf ${JAEGER_FILE}
rm ${JAEGER_FILE}.tar.gz

cd ~
mkdir otelcol
cd otelcol
wget https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v${OTELCOL_VER}/${OTELCOL_FILE}.tar.gz
tar xvzf ${OTELCOL_FILE}.tar.gz
sudo cp ./otelcol /usr/local/bin
cd ~
rm -rf otelcol


#sudo apt install -y make build-essential
#sudo wget https://golang.org/dl/${GO_FILE_NAME}
#sudo rm -rf /usr/local/go
#sudo tar -C /usr/local -xzf ${GO_FILE_NAME}
#rm -rf ${GO_FILE_NAME}
#
#export PATH=$PATH:/usr/local/go/bin:/home/ubuntu/go/bin
#echo 'export PATH=$PATH:/usr/local/go/bin:~/go/bin' >> ~/.bashrc
#
#git clone https://github.com/open-telemetry/opentelemetry-collector.git
#cd opentelemetry-collector
#git checkout v${OTELCOL_VER}
#make install-tools
#make otelcol
#sudo cp bin/otelcol_linux_amd64 /usr/local/bin/otelcol
#cd ~
#rm -rf opentelemetry-collector
