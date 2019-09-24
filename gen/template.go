/**
 * @version: 1.0.0
 * @author: zhangguodong:general_zgd
 * @license: LGPL v3
 * @contact: general_zgd@163.com
 * @site: github.com/generalzgd
 * @software: GoLand
 * @file: template.go
 * @time: 2019/8/8 21:22
 */
package gen

import (
	`bytes`
	"strconv"
	"strings"
	`text/template`

	`github.com/golang/protobuf/protoc-gen-go/generator`
	`github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/descriptor`
	`github.com/toolkits/slice`
)

const (
	TagTransmit = "@transmit"
	TagTarget   = "@target"
	TagId       = "@id"     // 上行请求协议对应的id
	TagUpId     = "@upid"   // 上行请求协议对应的id
	TagDownId   = "@downid" // 下行响应协议对应的id
)

type param struct {
	*descriptor.File
	Imports []descriptor.GoPackage
	// RegisterFunSuffix string
}

type defParam struct {
	*descriptor.File
	// Services []*descriptor.Service
	ServicesWithComment []*serviceWithComment
}

type serviceWithComment struct {
	*descriptor.Service
	MethodsWithComment []*methodWithComment
	Comment            string
	CommentList        []string
	TargetName         string // endpoint server
}

func (p *serviceWithComment) CanOutput() bool {
	for _, m := range p.MethodsWithComment {
		if m.CanOutput() {
			return true
		}
	}
	return false
}

func (p *serviceWithComment) ParseComment() {
	commentLines := strings.Split(p.Comment, "\n")
	for i, it := range commentLines {
		commentLines[i] = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(it), "//"))
	}
	p.CommentList = commentLines
}

func (p *serviceWithComment) GetFormatComment() string {
	if p.CommentList == nil {
		p.ParseComment()
	}
	li := make([]string, 0, len(p.CommentList))
	for _, line := range p.CommentList {
		li = append(li, "// "+line)
	}
	return strings.Join(li, "\n")
}

type methodWithComment struct {
	*descriptor.Method
	Comment     string
	CommentList []string
}

func (p *methodWithComment) ParseComment() {
	commentLines := strings.Split(p.Comment, "\n")
	for i, it := range commentLines {
		commentLines[i] = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(it), "//"))
	}
	p.CommentList = commentLines
}

func (p *methodWithComment) GetFormatComment() string {
	if p.CommentList == nil {
		p.ParseComment()
	}
	li := make([]string, 0, len(p.CommentList))
	for _, line := range p.CommentList {
		if strings.Contains(line, TagTransmit) || strings.Contains(line, TagTarget) ||
			strings.Contains(line, TagId) || strings.Contains(line, TagUpId) ||
			strings.Contains(line, TagDownId) {
			continue
		}
		li = append(li, "// "+line)
	}
	return strings.Join(li, "\n")
}

func (p *methodWithComment) CanOutput() bool {
	if p.CommentList == nil {
		p.ParseComment()
	}
	if !slice.ContainsString(p.CommentList, TagTransmit) {
		return false
	}
	return true
}

func (p *methodWithComment) GetTargetSvrName() string {
	if p.CanOutput() {
		tarCommentLine := ""
		for _, it := range p.CommentList {
			if strings.Contains(it, TagTarget) {
				tarCommentLine = it
				break
			}
		}

		if len(tarCommentLine) > 0 {
			tar := strings.TrimSpace(strings.TrimPrefix(tarCommentLine, TagTarget))
			tar = strings.TrimSpace(strings.Split(tar, " ")[0])
			if len(tar) > 0 {
				return generator.CamelCase(tar)
			}
		}
	}
	return *p.Service.Name // 默认返回当前服务名
}

func (p *methodWithComment) GetUpId() uint16 {
	if p.CanOutput() {
		idCommentLine := ""
		tag := TagId
		for _, it := range p.CommentList {
			if strings.Contains(it, TagUpId) {
				idCommentLine = it
				tag = TagUpId
				break
			} else if strings.Contains(it, TagId) {
				idCommentLine = it
				tag = TagId
				break
			}
		}

		if len(idCommentLine) > 0 {
			id := strings.TrimSpace(strings.TrimPrefix(idCommentLine, tag))
			id = strings.TrimSpace(strings.Split(id, " ")[0])
			if len(id) > 0 {
				v, _ := strconv.Atoi(id)
				return uint16(v)
			}
		}
	}
	return 0
}

func (p *methodWithComment) GetDownId() uint16 {
	if p.CanOutput() {
		idCommentLine := ""
		for _, it := range p.CommentList {
			if strings.Contains(it, TagDownId) {
				idCommentLine = it
				break
			}
		}
		if len(idCommentLine) > 0 {
			id := strings.TrimSpace(strings.TrimPrefix(idCommentLine, TagDownId))
			id = strings.TrimSpace(strings.Split(id, " ")[0])
			if len(id) > 0 {
				v, _ := strconv.Atoi(id)
				return uint16(v)
			}
		}
	}
	return 0
}

func applyTemplate(p param, name2Path map[string]string, path2Comment map[string]string) (string, error) {
	out := bytes.NewBuffer(nil)
	if err := headerTemplate.Execute(out, p); err != nil {
		return "", err
	}

	getComment := func(keys ...string) string {
		comment := ""
		pt := strings.Join(keys, "/")
		if path, ok := name2Path[pt]; ok {
			comment = path2Comment[path]
		}
		return comment
	}

	for _, msg := range p.Messages {
		msgName := generator.CamelCase(*msg.Name)
		msg.Name = &msgName
	}
	// 需要输出的服务
	var outServices []*serviceWithComment
	for _, svc := range p.Services {
		//
		svcName := generator.CamelCase(*svc.Name)
		svc.Name = &svcName

		svcIt := &serviceWithComment{
			Service: svc,
			Comment: getComment(*p.Name, *svc.Name),
		}
		svcIt.ParseComment()
		for _, meth := range svc.Methods {
			methName := generator.CamelCase(*meth.Name)
			meth.Name = &methName
			mIt := &methodWithComment{
				Method:  meth,
				Comment: getComment(*p.Name, *svc.Name, *meth.Name),
			}
			mIt.ParseComment()
			svcIt.MethodsWithComment = append(svcIt.MethodsWithComment, mIt)
		}

		if svcIt.CanOutput() {
			outServices = append(outServices, svcIt)
		}
	}

	// 根据 @target 目的服务，重新进行分类
	tmpServices := map[string]*serviceWithComment{}
	for _, svc := range outServices {
		for _, m := range svc.MethodsWithComment {
			if !m.CanOutput() {
				continue
			}
			tarName := m.GetTargetSvrName()
			tmpSvr, ok := tmpServices[tarName]
			if !ok {
				tmpSvr = &serviceWithComment{
					Service:    svc.Service,
					Comment:    svc.Comment,
					TargetName: tarName,
				}
				tmpServices[tarName] = tmpSvr
			}
			tmpSvr.MethodsWithComment = append(tmpSvr.MethodsWithComment, m)
		}
	}

	var tarServices []*serviceWithComment
	for _, svc := range tmpServices {
		tarServices = append(tarServices, svc)
	}

	def := defParam{
		File:                p.File,
		ServicesWithComment: tarServices,
	}
	if err := defTemplate.Execute(out, def); err != nil {
		return "", err
	}

	if err := transTamplate.Execute(out, def); err != nil {
		return "", err
	}

	return out.String(), nil
}

var (
	headerTemplate = template.Must(template.New("header").Parse(`
// Code generated by protoc-gen-grpc-tcpway. DO NOT EDIT.
// source: {{.GetName}}

/*
Package {{.GoPkg.Name}} is a tcp/ws proxy.

It translates protobuf/Json packet into gRPC APIs.
*/
package {{.GoPkg.Name}}

import (
	{{range $i := .Imports}}{{if $i.Standard}}{{$i | printf "%s\n"}}{{end}}{{end}}
	{{range $i := .Imports}}{{if not $i.Standard}}{{$i | printf "%s\n"}}{{end}}{{end}}
)
`))

	defTemplate = template.Must(template.New("def").Parse(`
// define func
type registerHandler func(args *TransmitArgs) (err error)

{{range $svc := .ServicesWithComment}}
type transmit_{{$svc.TargetName}}_Handler func(*TransmitArgs, {{$svc.TargetName}}Client) (proto.Message, error)
{{end}}

type TransmitArgs struct {
	Method      string
	Endpoint    string
	Conn        *grpc.ClientConn
	MD          metadata.MD
	Data        []byte
	Codec       uint16
	Opts        []grpc.DialOption
	DoneCallback func(proto.Message)
	ctx         context.Context
}

var (
	// tag @id to package.TargetService/Method map
	id2meth = map[uint16]string{
	{{range $svc := .ServicesWithComment}}{{range $m := $svc.MethodsWithComment}}
	{{$id := $m.GetUpId}}{{if ne $id 0}}{{$id}}: "{{$.GoPkg.Name}}.{{$svc.TargetName}}/{{$m.GetName}}",{{end}}{{end}}{{end}}
	}

	meth2id = map[string]uint16{}

	id2struct = map[uint16]proto.Message{
	{{range $svr := .ServicesWithComment}}{{range $m := $svr.MethodsWithComment}}
	{{$id := $m.GetUpId}}{{if ne $id 0}}{{$id}}:&{{$m.RequestType.GetName}}{},{{end}}
	{{$id := $m.GetDownId}}{{if ne $id 0}}{{$id}}:&{{$m.ResponseType.GetName}}{},{{end}}{{end}}{{end}}
	}

	structName2id = map[string]string{
	{{range $svr := .ServicesWithComment}}{{range $m := $svr.MethodsWithComment}}
	{{$id := $m.GetUpId}}{{if ne $id 0}}"{{$m.RequestType.GetName}}":{{$id}},{{end}}
	{{$id := $m.GetDownId}}{{if ne $id 0}}"{{$m.ResponseType.GetName}}":{{$id}},{{end}}{{end}}{{end}}
	}

	{{range $svc := .ServicesWithComment}}
	transmit_{{$svc.TargetName}}_Map = map[string]transmit_{{$svc.TargetName}}_Handler{}
	{{end}}

	serviceMap = map[string]registerHandler{}
)

func init() {
	for k, v := range id2meth {
		meth2id[v] = k
	}

	// todo something handler
	{{range $svc := .ServicesWithComment}}
		{{range $m := $svc.MethodsWithComment}}
	transmit_{{$svc.TargetName}}_Map["{{$.GoPkg.Name}}.{{$svc.TargetName}}/{{$m.GetName}}"] = request_{{$svc.TargetName}}_{{$m.GetName}}
		{{end}}
	{{end}}

	{{range $svc := .ServicesWithComment}}
	serviceMap["{{$.GoPkg.Name}}.{{$svc.TargetName}}"] = register_{{$svc.TargetName}}_Transmitor
	{{end}}
}

func decodeBytes(data []byte, codec uint16, inst proto.Message) error {
	if codec == 0 {
		if err := proto.Unmarshal(data, inst); err != nil {
			return err
		}
	} else if codec == 1 {
		if err := json.Unmarshal(data, inst); err != nil {
			return err
		}
	}
	return nil
}

// get meth(package.TargetService/Method) by id(cmdid)
func GetMethById(id uint16) string {
	return id2meth[id]
}

func GetIdByMeth(meth string) uint16 {
	return meth2id[meth]
}

// 根据@id/@upid/@downid标签获取对应方法的请求参数对象
func GetMsgObjById(id uint16) proto.Message {
	v := id2struct[id]
	return v
}

func GetIdByMsgObj(obj proto.Message) uint16 {
	name := comm_libs.GetStructName(obj)
	return structName2id[name]
}

func ParseMethod(method string) (string, string, string, error) {
	method = strings.Trim(method, "/")
	dotIdx := strings.Index(method, ".")
	slashIdx := strings.Index(method, "/")
	if dotIdx < 1 || slashIdx < 1 || dotIdx > slashIdx {
		return "", "", "", errors.New("method must be type of 'package.ServiceName/Method'")
	}
	packageName := method[:dotIdx]
	serviceName := strings.Trim(method[dotIdx:slashIdx], ".")
	methodName := strings.Trim(method[slashIdx:], "/")
	return packageName, serviceName, methodName, nil
}

// define call enter point
func RegisterTransmitor(args *TransmitArgs) error {
	if len(args.Method) < 1 || len(args.Endpoint) < 1 || len(args.MD) < 1 || args.DoneCallback == nil {
		return errors.New("transmit args empty")
	}

	packageName, serviceName, _, err := ParseMethod(args.Method)
	if err != nil {
		return err
	}
	packageService := packageName + "." + serviceName
	if handler, ok := serviceMap[packageService]; ok {
		err := handler(args)
		return err
	}
	return errors.New("method not register yet")
}

`))

	transTamplate = template.Must(template.New("meth").Parse(`
// registor single service enter point

{{range $svc := .ServicesWithComment}}
// *********************************************************************************
// 注册{{$svc.GetName}}传输转换入口
{{if $svc.Comment}}{{$svc.GetFormatComment}}{{end}}
func register_{{$svc.TargetName}}_Transmitor(args *TransmitArgs) (err error) {
	if args.Conn == nil {
		conn, err := grpc.Dial(args.Endpoint, args.Opts...)
		if err != nil {
			return err
		}
		defer conn.Close()
		args.Conn = conn
	}
	
	ctx := context.Background()
	ctx, _ = context.WithTimeout(ctx, 5*time.Second)
	ctx = metadata.NewOutgoingContext(ctx, args.MD)
	args.ctx = ctx
	//
	client := New{{$svc.TargetName}}Client(args.Conn)
	handler, ok := transmit_{{$svc.TargetName}}_Map[args.Method]
	if !ok {
		return errors.New("method error")
	}
	res, err := handler(args, client)
	if err != nil {
		return err
	}
	args.DoneCallback(res)
	return nil
}

{{range $m := $svc.MethodsWithComment}}
// 注册{{$svc.TargetName}}/{{$m.GetName}} 传输方法入口
{{if $m.Comment}}{{$m.GetFormatComment}}{{end}}
func request_{{$svc.TargetName}}_{{$m.GetName}}(args *TransmitArgs, client {{$svc.TargetName}}Client) (proto.Message, error) {
	protoReq := &{{$m.RequestType.GetName}}{}
	if err := decodeBytes(args.Data, args.Codec, protoReq); err != nil {
		return nil, errors.New("codec err["+err.Error()+"]")
	}
	reply, err := client.{{$m.GetName}}(args.ctx, protoReq)
	if err != nil {
		return nil, errors.New("call err["+err.Error()+"]")
	}
	return reply, nil
}
{{end}}
{{end}}
`))
)
