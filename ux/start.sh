#!/usr/bin/env bash

cd $(dirname $0)

./crudbox -port 9999 -token "$(cat ../testtoken)"
