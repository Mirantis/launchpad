#!/bin/sh

set -e

cd test
rm -f ./id_rsa_mcc
ssh-keygen -t rsa -f ./id_rsa_mcc -N ""

envsubst < cluster.yaml.tpl > cluster.yaml
envsubst < footloose.yaml.tpl > footloose.yaml

curl -L https://github.com/weaveworks/footloose/releases/download/0.6.3/footloose-0.6.3-linux-x86_64 > ./footloose
chmod +x ./footloose
./footloose create

set +e
../bin/mcc --debug install
result=$!

./footloose delete
docker volume prune -f

exit $!
