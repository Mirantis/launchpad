#!/bin/bash
echo "Spinning machines"
mykey=`cat ~/.ssh/id_rsa.pub`
echo "  - " $mykey >> cloud-init.yaml
multipass launch 18.04 -n manager --mem 4G --disk 10G --cloud-init cloud-init.yaml
multipass launch 18.04 -n worker  --mem 4G --disk 10G --cloud-init cloud-init.yaml
manager=`multipass info manager  --format json | jq .info.manager.ipv4[0]`
sed -i "s/\"192.168.64.10\"/$manager/g" launchpad.yml 

worker=`multipass info worker  --format json | jq .info.worker.ipv4[0]`
sed -i "s/\"192.168.64.11\"/$worker/g" launchpad.yml 

# make build  -C ../../
 ../../bin/launchpad apply -c ./launchpad.yaml
