#!/bin/bash
export HOST_IP=$(hostname -i | awk '{print $1}')
docker-compose pull
docker-compose up -d