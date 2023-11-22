CUR_DIR=$(shell pwd)

ifeq ($(VERSION),)
	VERSION=1.0.0
endif

.PHONY: all
all: build

# 清理
.PHONY: clean
clean:
	rm -rf ./bin/*
	rm -rf ./protocol/proto/*
	rm -rf ./gate/client_proto/client_proto_gen.go
	rm -rf ./gs/api/*.pb.go && rm -rf ./node/api/*.pb.go

# 构建服务器二进制文件
.PHONY: build
build:
	mkdir -p bin && go build -ldflags "-X main.VERSION=$(VERSION)" -o ./bin/ ./cmd/...

# 清理镜像
.PHONY: docker_clean
docker_clean:
	rm -rf ./docker/node/bin/node
	rm -rf ./docker/dispatch/bin/dispatch
	rm -rf ./docker/gate/bin/gate
	rm -rf ./docker/multi/bin/multi
	rm -rf ./docker/gs/bin/gs
	rm -rf ./docker/gm/bin/gm
	docker rmi flswld/node:$(VERSION)
	docker rmi flswld/dispatch:$(VERSION)
	docker rmi flswld/gate:$(VERSION)
	docker rmi flswld/multi:$(VERSION)
	docker rmi flswld/gs:$(VERSION)
	docker rmi flswld/gm:$(VERSION)

# 复制配置模板等文件
.PHONY: docker_config
docker_config:
	mkdir -p ./docker && cp -rf ./docker-compose.yaml ./docker/
	mkdir -p ./docker/node/bin && cp -rf ./cmd/node/* ./docker/node/bin/ && rm -rf ./docker/node/bin/*.go
	mkdir -p ./docker/dispatch/bin && cp -rf ./cmd/dispatch/* ./docker/dispatch/bin/ && rm -rf ./docker/dispatch/bin/*.go
	mkdir -p ./docker/gate/bin && cp -rf ./cmd/gate/* ./docker/gate/bin/ && rm -rf ./docker/gate/bin/*.go
	mkdir -p ./docker/multi/bin && cp -rf ./cmd/multi/* ./docker/multi/bin/ && rm -rf ./docker/multi/bin/*.go
	mkdir -p ./docker/gs/bin && cp -rf ./cmd/gs/* ./docker/gs/bin/ && rm -rf ./docker/gs/bin/*.go
	mkdir -p ./docker/gm/bin && cp -rf ./cmd/gm/* ./docker/gm/bin/ && rm -rf ./docker/gm/bin/*.go

# 构建镜像
.PHONY: docker_build
docker_build:
	mkdir -p ./docker/node/bin && cp -rf ./bin/node ./docker/node/bin/
	mkdir -p ./docker/dispatch/bin && cp -rf ./bin/dispatch ./docker/dispatch/bin/
	mkdir -p ./docker/gate/bin && cp -rf ./bin/gate ./docker/gate/bin/
	mkdir -p ./docker/multi/bin && cp -rf ./bin/multi ./docker/multi/bin/
	mkdir -p ./docker/gs/bin && cp -rf ./bin/gs ./docker/gs/bin/
	mkdir -p ./docker/gm/bin && cp -rf ./bin/gm ./docker/gm/bin/
	docker build -t flswld/node:$(VERSION) ./docker/node
	docker build -t flswld/dispatch:$(VERSION) ./docker/dispatch
	docker build -t flswld/gate:$(VERSION) ./docker/gate
	docker build -t flswld/multi:$(VERSION) ./docker/multi
	docker build -t flswld/gs:$(VERSION) ./docker/gs
	docker build -t flswld/gm:$(VERSION) ./docker/gm

# 安装natsrpc生成工具
.PHONY: dev_tool
dev_tool:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1
	go install github.com/byebyebruce/natsrpc/cmd/protoc-gen-natsrpc@v0.5.5

# 生成natsrpc协议代码
.PHONY: gen_natsrpc
gen_natsrpc:
	protoc \
	--proto_path=gs/api \
	--go_out=paths=source_relative:gs/api \
	--natsrpc_out=paths=source_relative:gs/api \
	gs/api/*.proto
	protoc \
	--proto_path=node/api \
	--go_out=paths=source_relative:node/api \
	--natsrpc_out=paths=source_relative:node/api \
	node/api/*.proto

# 生成客户端协议代码
.PHONY: gen_proto
gen_proto:
	cd protocol/hk4e_proto && \
	rm -rf ./proto && mkdir -p proto && \
	protoc --proto_path=./ --go_out=paths=source_relative:./proto ./*.proto && \
	protoc --proto_path=./ --go_out=paths=source_relative:./proto ./cmd/*.proto && \
	protoc --proto_path=./ --go_out=paths=source_relative:./proto ./pb/*.proto && \
	protoc --proto_path=./ --go_out=paths=source_relative:./proto ./server_only/*.proto && \
	mv ./proto/cmd/* ./proto/ && rm -rf ./proto/cmd && \
	mv ./proto/pb/* ./proto/ && rm -rf ./proto/pb && \
	mv ./proto/server_only/* ./proto/ && rm -rf ./proto/server_only && \
	rm -rf ../proto && mkdir -p ../proto && mv ./proto/* ../proto/ && rm -rf ./proto && \
	cd ../../

# 生成客户端协议代理功能所需的代码
.PHONY: gen_client_proto
gen_client_proto:
	cd gate/client_proto && \
	rm -rf client_proto_gen.go && \
	go test -count=1 -v -run TestClientProtoGen . && \
	rm -rf proto/*.pb.go && \
	find proto -name '*.proto' | xargs -n 1 protoc --proto_path=proto --go_out=proto

.PHONY: test
test:
	cd tests && go test -v

.PHONY: fmt
fmt:
	go fmt ./...
