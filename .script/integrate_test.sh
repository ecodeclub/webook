#!/usr/bin/env bash

set -e
docker compose -f .script/integration_test_compose.yml down -v
docker compose -f .script/integration_test_compose.yml up -d
go test  ./...  -tags=e2e
docker compose -f .script/integration_test_compose.yml down -v
