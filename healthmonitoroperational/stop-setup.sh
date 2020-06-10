#!/bin/bash

docker-compose -f docker/docker-compose-elasticsearch.yml down
docker-compose -f docker/docker-compose-kafka.yml down
docker stop cassandra
