#!/bin/sh

IP=$1
echo "create redis cluster, host ip: ${IP}"

redis-cli -a 123456 --cluster create \
  "${IP}":6371 \
  "${IP}":6372 \
  "${IP}":6373 \
  "${IP}":6374 \
  "${IP}":6375 \
  "${IP}":6376 \
  --cluster-replicas 1
