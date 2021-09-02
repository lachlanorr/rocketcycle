GO_VER=1.17
FILE_NAME=go${GO_VER}.linux-amd64.tar.gz

sudo apt update
sudo apt update # needed, not sure why, but updates come down the second time
sudo apt upgrade

sudo apt install -y make \
     build-essential \
     protobuf-compiler \
     emacs-nox

sudo wget https://golang.org/dl/${FILE_NAME}
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf ${FILE_NAME}
export PATH=$PATH:/usr/local/go/bin

sudo mkdir /code
sudo chown -R ubuntu:ubuntu /code
cd /code
git clone https://github.com/lachlanorr/rocketcycle.git

cd rocketcycle
go install \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2 \
    google.golang.org/protobuf/cmd/protoc-gen-go \
    google.golang.org/grpc/cmd/protoc-gen-go-grpc

echo 'PATH=$PATH:/usr/local/go/bin:~/go/bin' >> ~/.bashrc

export PATH=$PATH:~/go/bin
make
