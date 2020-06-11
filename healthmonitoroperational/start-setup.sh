#!/bin/bash

echo -e "Deleting all pre-existent docker containers."
docker rm -f $(docker ps -a -q)

sleep 10s

echo -e "Starting Database Setup"
docker-compose -f docker/docker-compose.yml up -d
