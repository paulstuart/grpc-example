#!/usr/bin/env bash

curl -s 'http://localhost:3200/api/search?limit=10' | jq -r '.traces[] | "\(.rootTraceName) - \(.durationMs)ms"'
