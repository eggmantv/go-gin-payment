#!/bin/bash

# 这里部署必须使用root用户
SSH_USER=root
WWW_ROOT=/disk1/www/go-gin-payment

deploy() {
    local host

    host=$1

    echo "****deploying to $host****"

    echo uploading to $SSH_USER@$host:$WWW_ROOT
    scp docker-compose.yml docker.env start_prd.sh $SSH_USER@$host:$WWW_ROOT/
    echo starting service...
    ssh $SSH_USER@$host "\
        cd $WWW_ROOT;\
        sh start_prd.sh;"
    
    echo done!
}

deploy api.xxx.com
