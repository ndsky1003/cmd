#!/bin/bash
GOOS=linux GOARCH=amd64 go build .
scp filemgr zy@hk-dev:filemgr

# ssh hk-dev " \
# rm filemgr  \
# mv filemgr_new filemgr \
# ./filemgr --server --suris \":18085\" --client --curis \"127.0.0.1:18085\" --secret cc123 \
# "
