GO_VER=1.17
GO_FILE_NAME=go${GO_VER}.linux-amd64.tar.gz

PROTOC_VER=3.17.3
PROTOC_FILE_NAME=protoc-3.17.3-linux-x86_64.zip

sudo apt update
sudo apt update # needed, not sure why, but updates come down the second time
sudo apt upgrade

sudo apt install -y make \
     build-essential \
     cpanminus \
     perl \
     perl-doc \
     libdbd-pg-perl \
     postgresql-client \
     unzip \
     emacs-nox

sudo cpanm --quiet --notest App::Sqitch

sudo wget https://golang.org/dl/${GO_FILE_NAME}
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf ${GO_FILE_NAME}
export PATH=$PATH:/usr/local/go/bin

sudo mkdir /code
sudo chown -R ubuntu:ubuntu /code
mkdir /code/protobuf
cd /code/protobuf
wget https://github.com/protocolbuffers/protobuf/releases/download/v3.17.3/${PROTOC_FILE_NAME}
unzip ${PROTOC_FILE_NAME}
rm -f ${PROTOC_FILE_NAME}
export PATH=${PATH}:/code/protobuf/bin

cd /code
git clone https://github.com/lachlanorr/rocketcycle.git

cd rocketcycle
go install \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.0.1
go install \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.0.1
go install \
    google.golang.org/protobuf/cmd/protoc-gen-go@v1.26.0
go install \
    google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.0.1

echo 'PATH=$PATH:/code/protobuf/bin:/usr/local/go/bin:~/go/bin' >> ~/.bashrc

export PATH=$PATH:~/go/bin
make
