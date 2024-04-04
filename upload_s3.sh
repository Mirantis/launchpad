#!/bin/bash

TAG=$(git describe --tags --abbrev=0)

for file in dist/release/*; do
    aws s3 cp "$file" s3://get-mirantis.com/launchpad/$TAG/"$(basename "$file")"
done
