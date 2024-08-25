#!/usr/bin/env bash

docker run -d --name=llm-server \
 -p 8080:8080 \
 -v data:/go/src/aurora-llm/clove-db-aurora \
 -e AUTH_TOKEN="change your own token" \
 -e ASSISTANT_ID="change your own assistant id" \
 --restart always \
 docker.io/ban11111/aurora-llm:v0.0.1