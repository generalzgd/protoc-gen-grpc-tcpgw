# grpc-tcpgw(tcp网关)代码生成插件

## Useage

### 插件文件

```shell
# 生成文件
go build -o $GOPATH/bin/protoc-gen-grpc-tcpgw
```



### 插件参数

```shell
# import_prefix: 导入包的前缀，默认空
# import_path: 导入包的指定目录，默认空（即要导入的包都在同一级目录里）
# file: 指定文件，默认空（由protoc传入），对应的文件要对应CodeGeneratorRequest结构
```

### 使用命令

```shell
# generate tcp/ws grpc gateway
protoc -Iproto --grpc-tcpgw_out=logtostderr=true:./goproto ./proto/imgate.proto
# -Iproto 增加一个导入目录
# --{grpc-tcpgw}_out 对应插件文件名 protoc-gen-{grpc-tcpgw}
# 对应生成 xxx.pb.tcpgw.go
```

## Schema

```protobuf
// 后端服务 authorize.proto
service Authorize {
    // 校验用户
    rpc Login (ImLoginRequest) returns (ImLoginReply) {}
}

// 后端服务 im.proto
service Im {
    // 已读
    rpc Read(ImReadRequest) returns (ImReadReply) {}
}

// 网关 imgate.proto
// 定义一个grpc tcp/ws网关
service ImGate {
	// 登录注释
	// @transmit
	// @target Authorize
	// @id 1
    rpc Login (ImLoginRequest) returns (ImLoginReply) {}
    
    // 已读
    // @transmit
    // @target Im
    // @id 2
    rpc Read(ImReadRequest) returns (ImReadReply) {}
}

// @transmit 识别需要转发的method(rpc)
// @target 目标后端服务名（一定要跟后端的服务名称对上），如果不存在则以当前service名代替（实际运行会有问题）
// 因此，对于该插件必须要有这两个tag，缺一不可
// 调用方法名、参数、返回类型也要跟后端服务的方法名、参数、返回类型对上
// @id 数字id与当前的对应方法(packet.Service/Method)一一绑定，可不重复。需要自己维护id
```

## 应用代码

```go
type GateClientPackHead struct {
	Length uint16 // body的长度，65535/1024 ~ 63k
	Seq    uint16 // 序列号
	Cmdid  uint16 // 协议id，可以映射到对应的service:method
	Ver    uint16 // 协议更新版本号 1.0.1 => 1*100 + 0*10 + 1 => 101
	Codec  uint16 // 0:proto  1:json
	Opt    uint16 // 备用字段
}

// 网关包, 小端
type GateClientPack struct {
	GateClientPackHead
	Body []byte // protobuf or json
}

// *****************************************************************************************
import (
	zqproto `.../grpc-proto/goproto`
)

// 转换协议并发送, 前提是解析出当前的包
func (p *Manager) translatePack(session *link.Session, pack *gatepack.GateClientPack, info *common.ClientConnInfo) error {
    // 根据cmdid映射，得到对应的后端地址
	address, ok := p.getCallEndpoint(pack.Cmdid)
    // 根据cmdid映射，得到对应的后端方法名称 package.Service/Method, 例如：zqproto.Authorize/Login
	method, ok := p.getCallMethod(pack.Cmdid)
	if !ok {
		return define.CmdidError
	}
	// 转换用户链接信息为metadata.MD，用于grpc的header传输。
    // 接收方将header信息转换成ClientConnInfo结构体，以获得用户链接信息
	md, err := p.ClientInfoToMD(info)
	if err != nil {
		return err
	}
	// grpc传输结束时的方法调用
	doneHandler := func(reply proto.Message) {
		p.sendReplyPack(session, pack, reply)
	}
	// 将pack的信息，转换传输给后端的服务
	if err := zqproto.RegisterTransmitor(address, method, md, pack.Body, pack.Codec, doneHandler, grpc.WithInsecure()); err != nil {
		return err
	}

	logs.Debug("start call grpc:", address)
	return nil
}
```
## 特点

```
1. 客户端不用关心后端服务有哪些，只需知道网关地址。由网关根据包头信息自动路由到后端服务并返回对应数据。
2. 支持双向数据发送
3. 同时支持protobuf和json两种协议格式
4. 对比grpc-ecosystem/grpc-gateway
4.1 ecosystem需要为每个后端服务都注册一个网关地址和端口，客户端需要关心对应服务的网关和端口。
4.2 ecosystem只支持http的短连接访问，不支持双向数据发送。
```





