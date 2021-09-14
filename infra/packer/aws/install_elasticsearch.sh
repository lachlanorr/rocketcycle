ES_VER=7.14.1
ES_FILE=elasticsearch-${ES_VER}-darwin-x86_64.tar.gz
ES_URL=https://artifacts.elastic.co/downloads/elasticsearch/${ES_FILE}

# As per https://www.elastic.co/guide/en/elasticsearch/reference/current/deb.html
wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | sudo apt-key add -
sudo apt-get install apt-transport-https
echo "deb https://artifacts.elastic.co/packages/7.x/apt stable main" | sudo tee /etc/apt/sources.list.d/elastic-7.x.list
sudo apt-get update && sudo apt-get install elasticsearch
