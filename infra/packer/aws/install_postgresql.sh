GO_VER=1.17
FILE_NAME=go${GO_VER}.linux-amd64.tar.gz

sudo apt update
sudo apt update # needed, not sure why, but updates come down the second time
sudo apt upgrade

sudo apt install -y postgresql-12
