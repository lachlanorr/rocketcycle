CONSUL_VER=1.11.2
CONSUL_ZIP=consul_${CONSUL_VER}_linux_amd64.zip
NOMAD_VER=1.2.3
NOMAD_ZIP=nomad_${NOMAD_VER}_linux_amd64.zip

sudo apt-get install unzip

# prep hashi user, used for consul and nomad
sudo useradd hashi -m
sudo usermod --shell /bin/bash hashi
echo -e "hashi:hashi" | sudo chpasswd
sudo usermod -aG sudo hashi

cd ~

wget https://releases.hashicorp.com/consul/${CONSUL_VER}/${CONSUL_ZIP}
unzip ${CONSUL_ZIP}
sudo chown root:root ./consul
sudo mv ./consul /usr/local/bin/
rm ${CONSUL_ZIP}

wget https://releases.hashicorp.com/nomad/${NOMAD_VER}/nomad_${NOMAD_VER}_linux_amd64.zip
unzip ${NOMAD_ZIP}
sudo chown root:root ./nomad
sudo mv ./nomad /usr/local/bin/
rm ${NOMAD_ZIP}
