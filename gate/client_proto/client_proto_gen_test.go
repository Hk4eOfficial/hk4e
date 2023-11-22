package client_proto

import (
	"os"
	"strings"
	"testing"
)

func TestClientProtoGen(t *testing.T) {
	// 生成根据proto类名获取对象实例的switch方法
	dir, err := os.ReadDir("./proto")
	if err != nil {
		panic(err)
	}
	protoObjNameList := make([]string, 0)
	for _, entry := range dir {
		if entry.IsDir() {
			continue
		}
		split := strings.Split(entry.Name(), ".")
		if len(split) < 2 || split[len(split)-1] != "proto" {
			continue
		}
		protoObjNameList = append(protoObjNameList, split[len(split)-2])
	}
	// 生成初始化cmdId和cmdName的方法
	clientCmdFile, err := os.ReadFile("./proto/client_cmd.csv")
	if err != nil {
		panic(err)
	}
	clientCmdData := string(clientCmdFile)
	clientCmdLineList := strings.Split(clientCmdData, "\n")
	// 生成代码文件
	fileData := "package client_proto\n"
	fileData += "\n"
	fileData += "import (\n"
	fileData += "\t\"hk4e/gate/client_proto/proto\"\n"
	fileData += ")\n"
	fileData += "\n"
	fileData += "func (c *ClientCmdProtoMap) LoadClientCmdIdAndCmdName() {\n"
	for _, clientCmdLine := range clientCmdLineList {
		// 清理空格以及换行符之类的
		clientCmdLine = strings.TrimSpace(clientCmdLine)
		if clientCmdLine == "" {
			continue
		}
		item := strings.Split(clientCmdLine, ",")
		if len(item) != 2 {
			panic("parse client cmd file error")
		}
		cmdName := item[0]
		cmdId := item[1]
		fileData += "\tc.clientCmdIdCmdNameMap[uint16(" + cmdId + ")] = \"" + cmdName + "\"\n"
		fileData += "\tc.clientCmdNameCmdIdMap[\"" + cmdName + "\"] = uint16(" + cmdId + ")\n"
	}
	fileData += "}\n"
	fileData += "\n"
	fileData += "func (c *ClientCmdProtoMap) GetClientProtoObjByName(protoObjName string) any {\n"
	fileData += "\tswitch protoObjName {\n"
	for _, protoObjName := range protoObjNameList {
		fileData += "\tcase \"" + protoObjName + "\":\n\t\treturn new(proto." + protoObjName + ")\n"
	}
	fileData += "\tdefault:\n"
	fileData += "\t\treturn nil\n"
	fileData += "\t}\n"
	fileData += "}\n"
	err = os.WriteFile("./client_proto_gen.go", []byte(fileData), 0644)
	if err != nil {
		panic(err)
	}
}
