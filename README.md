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
// 后端服务 backendsvr1.proto
service BackendSvr1 {
    // 方法1
    rpc Method1 (Method1Request) returns (Method1Reply) {}
}

// 后端服务 backendsvr2.proto
service BackendSvr2 {
    // 方法2
    rpc Method2(Method2Request) returns (Method2Reply) {}
}

// tcp网关 tcpgate.proto
// 定义一个grpc tcp/ws网关
service TcpGate {
	// 路由转发方法1
	// @transmit
	// @target BackendSvr1
	// @upid 1 请求协议(Method1Request)对应id, id唯一
	// @downid 2 响应协议(Method1Reply)对应id
    rpc Method1 (Method1Request) returns (Method1Reply) {}
    
    // 已读
    // @transmit
    // @target BackendSvr2
    // @upid 3 请求协议(Method1Request)对应id, id唯一
    // @downid 4 响应协议(Method1Reply)对应id
    rpc Method2(Method2Request) returns (Method2Reply) {}
}

// @transmit 识别需要转发的method(rpc)
// @target 目标后端服务名（一定要跟后端的服务名称对上），如果不存在则以当前service名代替（实际运行会有问题）
// @upid 数字id与当前的对应方法(packet.Service/Method)一一绑定，可不重复; 同时与请求方法参数（Method1Request）相绑定
// @downid 数字id与请求方法的响应参数（Method1Reply）相绑定。可不重复
// 因此，对于该插件必须要有以上四个tag，缺一不可
// 调用方法名、参数、返回类型也要跟后端服务的方法名、参数、返回类型对上
```

## 应用代码

```go
type GateClientPackHead struct {
	Length uint16 // body的长度，65535/1024 ~ 63k
	Seq    uint16 // 序列号
	Id     uint16 // 协议id，可以映射到对应的service:method
	Codec  uint16 // 0:proto  1:json
}

// 网关包, 小端
type GateClientPack struct {
	GateClientPackHead
	Body []byte // protobuf or json
}

// *****************************************************************************************
import (
    `github.com/generalzgd/link`
    `github.com/generalzgd/grpc-tcp-gateway/codec`
    `github.com/generalzgd/grpc-svr-frame/common`
    `github.com/astaxie/beego/logs`
	gwproto `github.com/generalzgd/grpc-tcp-gateway-proto/goproto`
    grpcpool `github.com/processout/grpc-go-pool`
)

// 转换协议并发送, 前提是解析出当前的包
func (p *Manager) translatePack(session *link.Session, pack *codec.GateClientPack, info *common.ClientConnInfo) error {
    // 根据cmdid映射，得到对应的后端方法名称 package.Service/Method, 例如：ZQProto.Authorize/Login
	meth := gwproto.GetMethById(pack.Id)
	if len(meth) < 1 {
		err = codec.IdFieldError
		return
	}
    // 获取后端地址的配置参数
	cfg, ok := p.getEndpointByMeth(meth)
	if !ok {
		err = codec.EndpointError
		return
	}
	// 转换用户链接信息为metadata.MD，用于grpc的header传输。
    // 接收方将header信息转换成ClientConnInfo结构体，以获得用户链接信息，
    // 可根据需要添加其他数据
	md, err := p.ClientInfoToMD(info)
	if err != nil {
		return err
	}
	// grpc传输结束时的方法调用
	doneHandler := func(reply proto.Message) {
		p.sendReplyPack(session, pack, reply)
	}
	// 将pack的信息，转换传输给后端的服务
	var conn *grpcpool.ClientConn
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
    // 从连接池里获取一个链接
	conn, err = p.GetGrpcConnWithLB(cfg, ctx)
	if err != nil {
		return
	}
    // 把链接还给连接池
	defer conn.Close()

	args := &gwproto.TransmitArgs{
		Method:       meth,
		Endpoint:     cfg.Address,
		Conn:         conn.ClientConn,
		MD:           md,
		Data:         pack.Body,
		Codec:        pack.Codec,
		DoneCallback: doneHandler,
		Opts:         nil,
	}
	// 将pack的信息，转换传输给后端的服务
	if err = gwproto.RegisterTransmitor(args); err != nil {
		return
	}
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

## 相关仓库地址

```
https://github.com/generalzgd/grpc-tcp-gateway
https://github.com/generalzgd/protoc-gen-grpc-tcpgw
https://github.com/generalzgd/grpc-tcp-gateway-proto
```

## PS

```
目前该项目处于试运行阶段，尚有不足之处。恳请广大网友提点迷津。
```





