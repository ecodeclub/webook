#!/bin/sh

# shellcheck disable=SC2044
for item in $(find . -type f -name '*.go' -not -path './.idea/*'); do
  goimports -l -w "$item";
done