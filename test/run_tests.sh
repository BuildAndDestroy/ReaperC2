#!/bin/bash

# This is what we want. Build a client to inject data
# Start mongodb, add in our data using the mongoclient
# Run a few api tests, then kill the server by running stop
docker build -t mongoclient -f mongoclient.dockerfile .
docker run --rm -it \
    -n mongodb \
    -p 27017:27017 \
    -d mongodb/mongodb-community-server:latest
docker run --rm -it mongoclient
docker container stop mongodb
