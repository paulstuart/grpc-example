#!/usr/bin/env bash

curl -s 'http://localhost:3200/api/search/tag/service.name/values' | jq -r '.tagValues[]'
