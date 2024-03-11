#!/usr/bin/env bash

set -e
docker compose -p webook -f .script/integration_test_compose.yml down -v
docker compose -p webook -f .script/integration_test_compose.yml up -d
go test -race -coverprofile=cover.out ./...  -tags=e2e
docker compose -p webook -f .script/integration_test_compose.yml down -v
