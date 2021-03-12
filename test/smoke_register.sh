#!/bin/bash

set -e

cd test
. ./smoke.common.sh

function cleanupuserfile() {
  rm -f ~/.mirantis-lanchpad/user.yaml
}

trap cleanupuserfile EXIT

unset ACCEPT_LICENSE
${LAUNCHPAD} register --name "Automation" --company "Test" --email "testing@example.com" --accept-license

grep -q "name: Automation" ~/.mirantis-launchpad/user.yaml
grep -q "company: Test" ~/.mirantis-launchpad/user.yaml
grep -q "email: testing@example.com" ~/.mirantis-launchpad/user.yaml
grep -q "eula: true" ~/.mirantis-launchpad/user.yaml
