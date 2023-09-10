#!/usr/bin/env bash

set -e
docker compose -f script/integration_test_compose.yml down
docker compose -f script/integration_test_compose.yml up -d
echo "127.0.0.1 slave.a.com" >> /etc/hosts
go test -race ./... -tags=e2e
docker compose -f script/integration_test_compose.yml down
