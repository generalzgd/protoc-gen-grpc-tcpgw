/**
 * @version: 1.0.0
 * @author: zgd: general_zgd
 * @license: LGPL v3
 * @contact: general_zgd@163.com
 * @site: github.com/generalzgd
 * @software: GoLand
 * @file: registry.go
 * @time: 2019-08-09 11:33
 */

package gen

import (
	"fmt"
	"log"

	descriptor2 "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/descriptor"
)

type Registry struct {
	*descriptor.Registry
	fileComments map[string]map[string]string
	commentsMap  map[string]string
}

func NewRegistry() *Registry {
	return &Registry{
		Registry:     descriptor.NewRegistry(),
		fileComments: map[string]map[string]string{},
		commentsMap:  map[string]string{},
	}
}

func (p *Registry) AddComments(key string, comments map[string]string) {
	p.fileComments[key] = comments
}

const (
	// packagePath = 2 //
	messagePath = 4 // message type
	// enumPath = 5
	servicePath = 6 // service type
	//
	// messageFieldPath = 2 // message field path
	methodPath = 2 // service rpc path
)

// map[
// 4,0:path:4 path:0 span:3 span:0 span:5 span:1 leading_comments:" FooRequest test\n"
// 4,1:path:4 path:1 span:8 span:0 span:10 span:1 leading_comments:" FooReply test\n"
// 6,0:path:6 path:0 span:13 span:0 span:16 span:1 leading_comments:" test comment\n"
// 6,0,2,0:path:6 path:0 path:2 path:0 span:15 span:4 span:43 leading_comments:"   send comment\n"
// ]
func (p *Registry) ParseCommentsSource(fileDesc []*descriptor2.FileDescriptorProto) error {
	p.commentsMap = make(map[string]string) // MessageName/ServiceName => path
	for i, desc := range fileDesc {
		log.Println(i, *desc.Name)
		//
		for i, msg := range desc.MessageType {
			pt := fmt.Sprintf("%d,%d", messagePath, i)
			p.commentsMap[*msg.Name] = pt
		}
		//
		for i, svr := range desc.Service {
			pt := fmt.Sprintf("%d,%d", servicePath, i)
			p.commentsMap[*desc.Name+"/"+*svr.Name] = pt
			for j, m := range svr.Method {
				pt := fmt.Sprintf("%d,%d,%d,%d", servicePath, i, methodPath, j)
				p.commentsMap[*desc.Name+"/"+*svr.Name+"/"+*m.Name] = pt
			}
		}
	}
	return nil
}
