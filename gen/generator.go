/**
 * @version: 1.0.0
 * @author: zhangguodong:general_zgd
 * @license: LGPL v3
 * @contact: general_zgd@163.com
 * @site: github.com/generalzgd
 * @software: GoLand
 * @file: generator.go.go
 * @time: 2019/8/8 13:46
 */
package gen

import (
	`fmt`
	`go/format`
	`log`
	`path`
	`path/filepath`
	`strings`

	`github.com/golang/protobuf/proto`
	plugingo `github.com/golang/protobuf/protoc-gen-go/plugin`
	`github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/descriptor`
	`github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/generator`
)

type pathType int

const (
	pathTypeImport pathType = iota
	pathTypeSourceRelative
)

type TcpGenerator struct {
	reg                *Registry
	baseImports        []descriptor.GoPackage
	registerFuncSuffix string
	pathType           pathType
}

func (p *TcpGenerator) Generate(targets []*descriptor.File) ([]*plugingo.CodeGeneratorResponse_File, error) {
	// panic("implement me")
	var files []*plugingo.CodeGeneratorResponse_File
	for _, file := range targets {
		code, err := p.generate(file)
		if err != nil {
			return nil, err
		}
		formatted, err := format.Source([]byte(code))
		if err != nil {
			return nil, err
		}
		name := file.GetName()
		if p.pathType == pathTypeImport && file.GoPkg.Path != "" {
			name = fmt.Sprintf("%s/%s", file.GoPkg.Path, filepath.Base(name))
		}
		ext := filepath.Ext(name)
		base := strings.TrimSuffix(name, ext)
		output := fmt.Sprintf("%s.pb.tcpgw.go", base)
		files = append(files, &plugingo.CodeGeneratorResponse_File{
			Name:    proto.String(output),
			Content: proto.String(string(formatted)),
		})
	}
	return files, nil
}

func (p *TcpGenerator) generate(file *descriptor.File) (string, error) {
	pkgSeen := make(map[string]bool)
	var imports []descriptor.GoPackage
	for _, pkg := range p.baseImports {
		pkgSeen[pkg.Path] = true
		imports = append(imports, pkg)
	}
	path2Comments := p.reg.fileComments[*file.Name]

	// outComments := map[string]string{}
	for _, svc := range file.Services {
		for _, m := range svc.Methods {
			imports = append(imports, p.addEnumPathParamImports(file, m, pkgSeen)...)

			pkg := m.RequestType.File.GoPkg
			if pkg == file.GoPkg || pkgSeen[pkg.Path] {
				continue
			}

			pkgSeen[pkg.Path] = true
			imports = append(imports, pkg)
		}
	}

	params := param{
		File:    file,
		Imports: imports,
		// RegisterFunSuffix: p.registerFuncSuffix,
	}
	return applyTemplate(params, p.reg.commentsMap, path2Comments)
}

// addEnumPathParamImports handles adding import of enum path parameter go packages
func (p *TcpGenerator) addEnumPathParamImports(file *descriptor.File, m *descriptor.Method, pkgSeen map[string]bool) []descriptor.GoPackage {
	var imports []descriptor.GoPackage
	for _, b := range m.Bindings {
		for _, pp := range b.PathParams {
			e, err := p.reg.LookupEnum("", pp.Target.GetTypeName())
			if err != nil {
				continue
			}
			pkg := e.File.GoPkg
			if pkg == file.GoPkg || pkgSeen[pkg.Path] {
				continue
			}
			pkgSeen[pkg.Path] = true
			imports = append(imports, pkg)
		}
	}
	return imports
}

func New(reg *Registry, registerFuncSuffix, pathTypeString string) generator.Generator {
	var imports []descriptor.GoPackage
	for _, pkgpath := range []string{
		"context",
		"encoding/json",
		"errors",
		"strings",
		"time",
		"github.com/generalzgd/comm-libs",
		"github.com/golang/protobuf/proto",
		"google.golang.org/grpc",
		"google.golang.org/grpc/metadata",
	} {
		pkg := descriptor.GoPackage{
			Path: pkgpath,
			Name: path.Base(pkgpath),
		}
		if err := reg.ReserveGoPackageAlias(pkg.Name, pkg.Path); err != nil {
			for i := 0; ; i++ {
				alias := fmt.Sprintf("%s_%d", pkg.Name, i)
				if err := reg.ReserveGoPackageAlias(alias, pkg.Path); err != nil {
					continue
				}
				pkg.Alias = alias
				break
			}
		}
		imports = append(imports, pkg)
	}

	var pathType pathType
	switch pathTypeString {
	case "", "import":
	case "source_relative":
		pathType = pathTypeSourceRelative
	default:
		log.Fatalf("Unknown path type %q: want 'import' or 'source_relative'", pathTypeString)
	}

	return &TcpGenerator{
		reg:                reg,
		baseImports:        imports,
		registerFuncSuffix: registerFuncSuffix,
		pathType:           pathType,
	}
}
