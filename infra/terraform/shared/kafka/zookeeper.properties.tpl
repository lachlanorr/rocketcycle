tickTime=2000
dataDir=/data/zookeeper
clientPort=2181
initLimit=5
syncLimit=2
4lw.commands.whitelist=*

%{ for i, ip in zookeeper_ips ~}
server.${i+1}=${ip}:2888:3888
%{ endfor ~}
