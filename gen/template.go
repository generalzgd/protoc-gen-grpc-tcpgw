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
	TagImport   = "@import"
	TagTransmit = "@transmit"
	TagTarget   = "@target"
	TagTarPkg   = "@tarpkg"
	TagId       = "@id"     // 上行请求协议对应的id
	TagUpId     = "@upid"   // 上行请求协议对应的id
	TagDownId   = "@downid" // 下行响应协议对应的id
)

type param struct {
	*descriptor.File
	Imports []descriptor.GoPackage
	// RegisterFunSuffix string
	WithTransmitArgs bool
	DefinePrefix     string
	AdditionImports     []string
}

type defParam struct {
	*descriptor.File
	// Services []*descriptor.Service
	ServicesWithComment []*serviceWithComment
	DefinePrefix        string
}

type serviceWithComment struct {
	*descriptor.Service
	MethodsWithComment []*methodWithComment
	Comment            string
	CommentList        []string
	TargetName         string // endpoint server
	TargetPkg          string // 目标服务所在的包
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

func (p *serviceWithComment) ParseAdditionalImport() []string {
	var list []string
	for _, line := range p.CommentList {
		if strings.Contains(line, TagImport) {
			tar := strings.TrimSpace(strings.TrimPrefix(line, TagImport))
			tar = strings.TrimSpace(strings.Split(tar, " ")[0])
			if len(tar) > 0 {
				tmp := strings.SplitN(tar, ":", 2)
				if len(tmp) < 2{
					continue
				}
				flag, _ := strconv.Atoi(tmp[1])
				if flag&1 > 0 {
					list = append(list, tmp[0])
				}
			}
		}
	}
	return list
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

func (p *methodWithComment) GetRequestPackage() string {
	pkg := p.RequestType.File.GetPackage()
	if len(pkg) > 0 {
		return pkg + "."
	}
	return ""
}

func (p *methodWithComment) GetResponsePackage() string {
	pkg := p.ResponseType.File.GetPackage()
	if len(pkg) > 0 {
		return pkg + "."
	}
	return ""
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

func (p *methodWithComment) GetTargetSvrPackage() string {
	if p.CanOutput() {
		tarPkg := ""
		for _, it := range p.CommentList {
			if strings.Contains(it, TagTarPkg) {
				tarPkg = it
				break
			}
		}
		if len(tarPkg) > 0 {
			tar := strings.TrimSpace(strings.TrimPrefix(tarPkg, TagTarPkg))
			tar = strings.TrimSpace(strings.Split(tar, " ")[0])
			if len(tar) > 0 {
				return tar + "."
			}
		}
	}
	return ""
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
	var addiImport []string
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
		for _, im := range svcIt.ParseAdditionalImport() {
			if !slice.ContainsString(addiImport, im) {
				addiImport = append(addiImport, im)
			}
		}
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

	p.AdditionImports = addiImport
	if err := headerTemplate.Execute(out, p); err != nil {
		return "", err
	}

	// 根据 @target 目的服务，重新进行分类
	tmpServices := map[string]*serviceWithComment{}
	for _, svc := range outServices {
		for _, m := range svc.MethodsWithComment {
			if !m.CanOutput() {
				continue
			}
			tarName := m.GetTargetSvrName()
			tarPkg := m.GetTargetSvrPackage()
			tmpSvr, ok := tmpServices[tarName]
			if !ok {
				tmpSvr = &serviceWithComment{
					Service:    svc.Service,
					Comment:    svc.Comment,
					TargetName: tarName,
					TargetPkg:  tarPkg,
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
		DefinePrefix:        p.DefinePrefix,
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
	
	{{range $i := .AdditionImports}}
		"{{$i}}"{{end}}
)
`))

	defTemplate = template.Must(template.New("def").Parse(`
// define func
{{$dot := "."}}
{{$prefix := .DefinePrefix}}
{{range $svc := .ServicesWithComment}}
type {{$prefix}}transmit_{{$svc.TargetName}}_Handler func(*{{$prefix}}TransmitArgs, {{$svc.TargetPkg}}{{$svc.TargetName}}Client) (proto.Message, error)
{{end}}

type {{$prefix}}registerHandler func(args *{{$prefix}}TransmitArgs) (err error)

type {{$prefix}}TransmitArgs struct {
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
	// definePrefix = {{.DefinePrefix}}
	// tag @id to package.TargetService/Method map
	{{$prefix}}id2meth = map[uint16]string{}

	{{$prefix}}meth2id = map[string]uint16{}

	{{$prefix}}id2struct = map[uint16]func()proto.Message{}

	{{$prefix}}structName2id = map[string]uint16{}

	{{range $svc := .ServicesWithComment}}
	{{$prefix}}transmit_{{$svc.TargetName}}_Map = map[string]{{$prefix}}transmit_{{$svc.TargetName}}_Handler{}
	{{end}}

	{{$prefix}}serviceMap = map[string]{{$prefix}}registerHandler{}
)

func init() {
	// id2meth
	{{range $svc := .ServicesWithComment}}
		{{range $m := $svc.MethodsWithComment}}
{{$id := $m.GetUpId}}{{if ne $id 0}}{{$prefix}}id2meth[{{$id}}] = "{{$.GoPkg.Name}}.{{$svc.TargetName}}/{{$m.GetName}}"{{end}}{{end}}
	{{end}}
	// meth2id
	for k, v := range {{$prefix}}id2meth {
		{{$prefix}}meth2id[v] = k
	}
	// id2struct
	{{$prefix}}id2struct[6172] = func()proto.Message{return &imdef.ImError{}} // todo 这行为工具写死的代码,应该改成模板
	{{$prefix}}id2struct[8197] = func()proto.Message{return &comm.HfError{}}
	{{range $svr := .ServicesWithComment}}
		{{range $m := $svr.MethodsWithComment}}
			{{$id := $m.GetUpId}}{{if ne $id 0}}{{$prefix}}id2struct[{{$id}}] = func()proto.Message{return &{{$m.GetRequestPackage}}{{$m.RequestType.GetName}}{}}{{end}}
			{{$id := $m.GetDownId}}{{if ne $id 0}}{{$prefix}}id2struct[{{$id}}] = func()proto.Message{return &{{$m.GetResponsePackage}}{{$m.ResponseType.GetName}}{}}{{end}}
		{{end}}
	{{end}}
	
	// structName2id
	{{$prefix}}structName2id["ImError"] = 6172 // todo 这行为工具写死的代码,应该改成模板
	{{$prefix}}structName2id["HfError"] = 8197
	{{range $svr := .ServicesWithComment}}
		{{range $m := $svr.MethodsWithComment}}
			{{$id := $m.GetUpId}}{{if ne $id 0}}{{$prefix}}structName2id["{{$m.RequestType.GetName}}"] = {{$id}}{{end}}
			{{$id := $m.GetDownId}}{{if ne $id 0}}{{$prefix}}structName2id["{{$m.ResponseType.GetName}}"] = {{$id}}{{end}}
		{{end}}
	{{end}}

	// todo something handler
	{{range $svc := .ServicesWithComment}}
		{{range $m := $svc.MethodsWithComment}}
	{{$prefix}}transmit_{{$svc.TargetName}}_Map["{{$.GoPkg.Name}}.{{$svc.TargetName}}/{{$m.GetName}}"] = {{$prefix}}request_{{$svc.TargetName}}_{{$m.GetName}}{{end}}
	{{end}}

	{{range $svc := .ServicesWithComment}}
	{{$prefix}}serviceMap["{{$.GoPkg.Name}}.{{$svc.TargetName}}"] = {{$prefix}}register_{{$svc.TargetName}}_Transmitor{{end}}
}

func {{$prefix}}DecodeBytes(data []byte, codec uint16, inst proto.Message) error {
	if codec == 0 {
		return proto.Unmarshal(data, inst)
	} else if codec == 1 {
		return json.Unmarshal(data, inst)
	}
	return errors.New("codec type error")
}

func {{$prefix}}EncodeBytes(codec uint16, inst proto.Message) ([]byte, error) {
	if codec == 0 {
		return proto.Marshal(inst)
	} else if codec == 1 {
		return json.Marshal(inst)
	}
	return nil, errors.New("codec type error")
}

// get meth(package.TargetService/Method) by id(cmdid)
func {{$prefix}}GetMethById(id uint16) string {
	return {{$prefix}}id2meth[id]
}

func {{$prefix}}GetIdByMeth(meth string) uint16 {
	return {{$prefix}}meth2id[meth]
}

// 根据@id/@upid/@downid标签获取对应方法的请求参数对象
func {{$prefix}}GetMsgObjById(id uint16) (proto.Message, bool) {
	if f, ok := {{$prefix}}id2struct[id]; ok {
		return f(), true
	}
	return nil, false
}

func {{$prefix}}GetIdByMsgObj(obj proto.Message) uint16 {
	name := comm_libs.GetStructName(obj)
	return {{$prefix}}structName2id[name]
}

func {{$prefix}}ParseMethod(method string) (string, string, string, error) {
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
func {{$prefix}}RegisterTransmitor(args *{{$prefix}}TransmitArgs) error {
	if len(args.Method) < 1 || len(args.Endpoint) < 1 || len(args.MD) < 1 || args.DoneCallback == nil {
		return errors.New("transmit args empty")
	}

	packageName, serviceName, _, err := {{$prefix}}ParseMethod(args.Method)
	if err != nil {
		return err
	}
	packageService := packageName + "." + serviceName
	if handler, ok := {{$prefix}}serviceMap[packageService]; ok {
		err := handler(args)
		return err
	}
	return errors.New("method not register yet")
}

`))

	transTamplate = template.Must(template.New("meth").Parse(`
// registor single service enter point
{{$prefix := .DefinePrefix}}
{{range $svc := .ServicesWithComment}}
// *********************************************************************************
// 注册{{$svc.GetName}}传输转换入口
{{if $svc.Comment}}{{$svc.GetFormatComment}}{{end}}
func {{$prefix}}register_{{$svc.TargetName}}_Transmitor(args *{{$prefix}}TransmitArgs) (err error) {
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
	client := {{$svc.TargetPkg}}New{{$svc.TargetName}}Client(args.Conn)
	handler, ok := {{$prefix}}transmit_{{$svc.TargetName}}_Map[args.Method]
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
func {{$prefix}}request_{{$svc.TargetName}}_{{$m.GetName}}(args *{{$prefix}}TransmitArgs, client {{$svc.TargetPkg}}{{$svc.TargetName}}Client) (proto.Message, error) {
	protoReq := &{{$m.GetRequestPackage}}{{$m.RequestType.GetName}}{}
	if err := {{$prefix}}DecodeBytes(args.Data, args.Codec, protoReq); err != nil {
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
