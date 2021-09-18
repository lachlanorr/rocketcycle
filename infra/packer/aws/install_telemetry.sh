JAEGER_VER=1.26.0
JAEGER_FILE=jaeger-${JAEGER_VER}-linux-amd64

OTELCOL_VER=0.35.0
OTELCOL_FILE=otelcol_${OTELCOL_VER}_linux_amd64

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
