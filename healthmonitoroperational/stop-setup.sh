#!/bin/bash

docker-compose -f docker/docker-compose-elasticsearch.yml down
docker stop cassandra
