# hk4e

## What's this

#### A Genshin Impact game server written in Golang.

#### This project aims to build a high performance ARPG game server, instead of supporting all in-game features

#### The network protocols and configuration tables are mainly from version 3.2, so we suggest you to use a v3.2 client, although it is not necessary

[Download version 3.2.0 from miHoYo](https://autopatchhk.yuanshen.com/client_app/download/pc_zip/20221024103618_h2e3o3zijYKEqHnQ/GenshinImpact_3.2.0.zip)

#### You need to use a https proxy and a patch to use it. Grasscutter is a good example.

#### You can use a newer patch to avoid using https proxies. It supports custom keys and multiple servers. Special thanks to [Jx2f/mhypbase](https://github.com/Jx2f/mhypbase)

## Features

* Native high-availability cluster architecture. Crashes on any node servers won't affect the whole system. So it is
  highly extendable
* Player-level non-status game server. Non-lock single thread structure. Easy to develop. And fully-featured player
  data exchanger(`Memory<->Cache<->Database`), say goodbye to
* synchronization-blocked database access
* Grand new cross-server player migration
* Gateway server side client protocol proxy conversion. Never worry about code changes caused by confusions of message
  number and protocol number
* Fully-featured key exchanging strategy with enhanced security, instead of using an existing random seed with an XOR
  key file all day long

## Compile and working environment

* Go >= 1.18
* Protoc >= 3.21
* Protoc Gen Go >= 1.28
* Docker >= 20.10
* Docker Compose >= 1.29

## Get start

* Install on first launch

```shell
make dev_tool
```

* Generate protocol

```shell
make gen_natsrpc      # Generate natsrpc protocol
make gen_proto        # Generate client protocol
make gen_client_proto # Generate client proxy protocol(not very necessary, for further information: gate/client_proto/README.md)
```

* Compile

```shell
make build         # Compile server to binary file
make docker_config # Copy files like config templates, etc.
make docker_build  # Build a Docker image
```

* Launch

```shell
cd docker
# Checkout all configurations before launch(e.g. docker/node/bin/application.toml)
docker-compose up -d # Launch server
```

#### Third-party dependencies

* mongodb
* nats-server
* redis

#### Server components

* `node` Node server (Single node, with status)
* `dispatch` Login server (Multi nodes, without status)
* `gate` Gateway server (Multi nodes, with status)
* `multi` Multi-function server (Multi nodes, with status **STILL UNDER CONSTRUCTION**)
* `gs` Game server (Multi nodes, with status)
* `gm` Game management server (Single node, without status)

#### Misc

* You need to add the following environment variable to run as standalone server

```shell
GOLANG_PROTOBUF_REGISTRATION_CONFLICT=ignore
```

## Code submission specifications

* You **MUST** run `go fmt` before commit
* Please ignore directory `gdconf` when running global formatting. These are config tables, including lots
  of `json`, `loa`, `txt` files
* Add a `space` between `//` and your comments when writing inline comments
* Split standard packages, local packages and third-party packages with linebreaks
