#!/usr/bin/env bash

set -e # Exit script immediately on first error.

docker stop llm-server && docker rm llm-server