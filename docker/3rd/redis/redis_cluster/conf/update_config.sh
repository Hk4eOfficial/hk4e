#!/bin/sh

IP=$1
echo "update redis config, host ip: ${IP}"

sed -i s/192.168.199.233/"${IP}"/ ./redis1/redis.conf
sed -i s/192.168.199.233/"${IP}"/ ./redis2/redis.conf
sed -i s/192.168.199.233/"${IP}"/ ./redis3/redis.conf
sed -i s/192.168.199.233/"${IP}"/ ./redis4/redis.conf
sed -i s/192.168.199.233/"${IP}"/ ./redis5/redis.conf
sed -i s/192.168.199.233/"${IP}"/ ./redis6/redis.conf
