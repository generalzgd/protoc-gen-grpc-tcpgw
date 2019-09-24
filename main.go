/**
 * @version: 1.0.0
 * @author: zhangguodong:general_zgd
 * @license: LGPL v3
 * @contact: general_zgd@163.com
 * @site: github.com/generalzgd
 * @software: GoLand
 * @file: main.go
 * @time: 2019/8/8 13:34
 */
package main

import (
	`flag`
	"fmt"
	`io/ioutil`
	`log`
	`os`
	`path/filepath`
	`runtime`
	"strconv"
	`strings`

	`github.com/golang/glog`
	`github.com/golang/protobuf/proto`
	plugin_go `github.com/golang/protobuf/protoc-gen-go/plugin`
	`github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway/descriptor`

	`github.com/generalzgd/protoc-gen-grpc-tcpgw/gen`
)

var (
	importPrefix = flag.String("import_prefix", "", "prefix to be added to go package paths for imported proto files")
	importPath   = flag.String("import_path", "", "used as the package if no input files declare go_package. If it contains slashes, everything up to the rightmost slash is ignored.")
	file         = flag.String("file", "-", "where to load data from")
	//file               = flag.String("file", "./test_in.bts", "where to load data from")
	registerFuncSuffix = flag.String("register_func_suffix", "Handler", "used to construct names of generated Register*<Suffix> methods.")
	pathType           = flag.String("paths", "", "specifies how the paths of generated files are structured")
	versionFlag        = flag.Bool("version", false, "print current version")
	debug              = flag.Bool("debug", false, "")
)

var (
	version = "1.0.1"
)

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Version %v", version)
		os.Exit(0)
	}

	reg := gen.NewRegistry()

	f := os.Stdin
	if *file != "-" {
		var err error
		pt := *file
		if runtime.GOOS == "windows" {
			pt = filepath.Join(filepath.Dir(os.Args[0]), strings.Trim(pt, "./"))
		}

		f, err = os.Open(pt)
		if err != nil {
			log.Fatal(err, "open file err.")
			return
		}
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err, "reading input")
	}

	req := new(plugin_go.CodeGeneratorRequest)
	if err := proto.Unmarshal(data, req); err != nil {
		log.Fatal(err, "parsing input data")
	}

	// log.Printf("input data:%v", g.Request)
	// jsonStr,_ := json.Marshal(g.Request)
	// ioutil.WriteFile("test_in.json", jsonStr, os.ModePerm)
	if *debug {
		ioutil.WriteFile("test_in.bts", data, os.ModePerm)
	}

	if len(req.FileToGenerate) == 0 {
		log.Fatal("no files to generate")
	}

	if req.Parameter != nil {
		list := strings.Split(req.GetParameter(), ",")
		for _, p := range list {
			spec := strings.SplitN(p, "=", 2)
			if len(spec) == 1 {
				if err := flag.CommandLine.Set(spec[0], ""); err != nil {
					log.Fatal("cannot set flag", p)
				}
				continue
			}
			name, value := spec[0], spec[1]
			if err := flag.CommandLine.Set(name, value); err != nil {
				log.Fatal("set falg fail.", p)
			}
		}
	}

	g := gen.New(reg, *registerFuncSuffix, *pathType)

	reg.SetPrefix(*importPrefix)
	reg.SetImportPath(*importPath)

	if err := reg.Load(req); err != nil {
		emitError(err)
	}

	var targets []*descriptor.File
	for _, target := range req.FileToGenerate {
		f, err := reg.LookupFile(target)
		if err != nil {
			log.Fatal(err)
		}
		targets = append(targets, f)
		//
		comments := extractComments(f)
		reg.AddComments(*f.Name, comments)
		// log.Println(comments)
	}

	if err := reg.ParseCommentsSource(req.ProtoFile); err != nil {
		emitError(err)
	}

	out, err := g.Generate(targets)
	if err != nil {
		emitError(err)
		return
	}

	emitFiles(out)
}

func extractComments(file *descriptor.File) map[string]string {
	comments := make(map[string]string)
	for _, loc := range file.GetSourceCodeInfo().GetLocation() {
		if loc.LeadingComments == nil {
			continue
		}
		var t []string
		for _, n := range loc.Path {
			t = append(t, strconv.Itoa(int(n)))
		}
		comments[strings.Join(t, ",")] = strings.TrimSpace(*loc.LeadingComments)
	}
	return comments
}

func emitFiles(out []*plugin_go.CodeGeneratorResponse_File) {
	emitResp(&plugin_go.CodeGeneratorResponse{File: out})
}

func emitError(err error) {
	emitResp(&plugin_go.CodeGeneratorResponse{Error: proto.String(err.Error())})
}

func emitResp(resp *plugin_go.CodeGeneratorResponse) {
	buf, err := proto.Marshal(resp)
	if err != nil {
		glog.Fatal(err)
	}
	if _, err := os.Stdout.Write(buf); err != nil {
		glog.Fatal(err)
	}

	if *debug {
		ioutil.WriteFile("test_out.pb.tcpgw.go", buf, os.ModePerm)
	}
}
