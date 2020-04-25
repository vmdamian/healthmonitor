#!/bin/bash

echo -e "Deleting all pre-existent docker containers."
docker rm -f $(docker ps -a -q)
echo -e "Starting Cassandra"
docker run --name cassandra --rm -p 9042:9042 -d -v /var/healthmonitor/cassandra/data:/var/lib/cassandra cassandra 
echo -e "Starting Elasticsearch"
docker-compose -f docker/docker-compose-elasticsearch.yml up -d

