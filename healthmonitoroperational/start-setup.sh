#!/bin/bash

echo -e "Deleting all pre-existent docker containers."
docker rm -f $(docker ps -a -q)

sleep 10s

echo -e "Starting Elasticsearch"
docker-compose -f docker/docker-compose-elasticsearch.yml up -d

sleep 10s

echo -e "Starting Kafka"
docker-compose -f docker/docker-compose-kafka.yml up -d

sleep 10s

echo -e "Starting Cassandra"
docker run --name cassandra --rm -p 9042:9042 -d -v /home/damian/go/src/healthmonitor/healthmonitordata/cassandra/data:/var/lib/cassandra cassandra 

