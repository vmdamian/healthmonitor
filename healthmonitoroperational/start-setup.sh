#!/bin/bash

echo -e "Deleting all pre-existent docker containers."
docker rm -f $(docker ps -a -q)

sleep 5s

echo -e "Starting Cassandra"
docker run --name cassandra --rm -p 9042:9042 -d -v /var/healthmonitor/cassandra/data:/var/lib/cassandra cassandra 

sleep 10s

echo -e "Starting Elasticsearch"
docker-compose -f docker/docker-compose-elasticsearch.yml up -d

sleep 4m

echo -e "Starting Healthmonitor API"
/home/$USER/go/src/healthmonitor/healthmonitorapi/healthmonitorapi &

sleep 10s

echo -e "Starting Healthmonitor UI"
cd /home/$USER/go/src/healthmonitor/healthmonitorui/ 
npm run serve
