#!/bin/bash

docker-compose -f docker-compose-elasticsearch.yml down
docker stop cassandra
