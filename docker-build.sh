#!/bin/bash
go mod vendor
docker build --platform linux/amd64 --build-arg APP_ROOT=/disk1/www/go-gin-payment -t registry.xxx.com/go-gin-payment .
rm -rf vendor

# push
push="docker push registry.xxx.com/go-gin-payment:latest"
echo $push
eval $push
