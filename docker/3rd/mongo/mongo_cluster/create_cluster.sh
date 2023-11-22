#!/bin/sh

IP=$1
echo "create mongo cluster, host ip: ${IP}"

echo "init mongo cluster shard"
# shard1
mongo --host "${IP}" --port 27118 <<EOF
rs.initiate({_id: "shard1", members: [{_id: 0, host: "mongo_shard1:27018"}]})
EOF
# shard2
mongo --host "${IP}" --port 27218 <<EOF
rs.initiate({_id: "shard2", members: [{_id: 0, host: "mongo_shard2:27018"}]})
EOF
# shard3
mongo --host "${IP}" --port 27318 <<EOF
rs.initiate({_id: "shard3", members: [{_id: 0, host: "mongo_shard3:27018"}]})
EOF

echo "init mongo cluster config"
# config1
mongo --host "${IP}" --port 27119 <<EOF
rs.initiate({_id: "config", configsvr: true, members: [{_id: 0, host: "mongo_config1:27019"}, {_id: 1, host: "mongo_config2:27019"}, {_id: 2, host: "mongo_config3:27019"}]})
EOF
sleep 30

echo "init mongo cluster mongos"
# mongos1
mongo --host "${IP}" --port 27117 <<EOF
sh.addShard("shard1/mongo_shard1:27018")
sh.addShard("shard2/mongo_shard2:27018")
sh.addShard("shard3/mongo_shard3:27018")
EOF
sleep 5

echo "init hk4e database table"
mongo --host "${IP}" --port 27117 <<EOF
sh.enableSharding("node_hk4e")
sh.shardCollection("node_hk4e.region", {"region_id": "hashed"})
sh.enableBalancing("node_hk4e.region")
sh.enableSharding("dispatch_hk4e")
sh.shardCollection("dispatch_hk4e.account", {"account_id": "hashed"})
sh.enableBalancing("dispatch_hk4e.account")
sh.shardCollection("dispatch_hk4e.client_log", {"_id": "hashed"})
sh.enableBalancing("dispatch_hk4e.client_log")
sh.enableSharding("gate_hk4e")
sh.shardCollection("gate_hk4e.account", {"open_id": "hashed"})
sh.enableBalancing("gate_hk4e.account")
sh.enableSharding("gs_hk4e")
sh.shardCollection("gs_hk4e.player", {"player_id": "hashed"})
sh.enableBalancing("gs_hk4e.player")
sh.shardCollection("gs_hk4e.chat_msg", {"uid": "hashed"})
sh.enableBalancing("gs_hk4e.chat_msg")
sh.startBalancer()
db.adminCommand("flushRouterConfig")
EOF
