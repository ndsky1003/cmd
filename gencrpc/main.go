package main

import (
	_ "embed"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

var (
	file_path string
	out_path  string
	suffix    string
	imports   string
)

// 格式化 Go 文件
func formatGoFile(filePath string) error {
	cmd := exec.Command("gofmt", "-w", filePath)
	return cmd.Run()
}

func main() {
	flag.String("useage", "useage", "gencrpc --sub=ccc")
	flag.StringVar(&file_path, "f", "", "需要解析的文件")
	flag.StringVar(&out_path, "o", "", "输出的新文件")
	flag.StringVar(&suffix, "sub", "_crpc_gen", "输出的新文件的后缀")
	flag.StringVar(&imports, "import", "", "额外需要导入的包，多个包用逗号分隔,别名用冒号分开。eg: --import=jj:encoding/json,tt:time")
	flag.StringVar(&data.ReqAppend, "req_append", "", "额外参数，多个参数用逗号分隔,别名用冒号分开。eg: --append_req=tt:time,opts:...crpc.Option")
	flag.StringVar(&data.Client, "client", "crpc_client", "生成客户端代码时使用的变量名")
	flag.StringVar(&data.Server, "server", "crpc_server_name", "调用哪个服务")
	flag.StringVar(&data.Module, "module", "crpc", "生成代码时使用的模块名")
	flag.Parse()
	if file_path == "" {
		dir, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		filename := os.Getenv("GOFILE")
		if filename == "" {
			return
		}
		file_path = filepath.Join(dir, filename)
	}
	dir, filename := filepath.Split(file_path)
	if !strings.HasSuffix(filename, ".go") {
		panic("源文件后缀必须是 .go")
	}
	if out_path == "" {
		filename = filename[:len(filename)-3]
		filename_new := fmt.Sprintf("%s%s.go", filename, suffix)
		out_path = filepath.Join(dir, filename_new)
	}

	fmt.Println("解析文件:", file_path)
	// 解析 Go 文件
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, file_path, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	// 获取包名
	data.PackageName = node.Name.Name

	// 遍历 AST
	ast.Inspect(node, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ImportSpec:
			if importor := handleImportSpec(n); importor != nil {
				data.Importor_all[importor.Name] = importor
			}
		case *ast.FuncDecl:
			if func_desc := handleFuncDecl(n); func_desc != nil {
				// fmt.Println("func_desc:", func_desc)
				data.Funcs = append(data.Funcs, func_desc)
			}
		}
		return true
	})
	data.FixImportorNeedList()
	tmpl, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	outFile, err := os.OpenFile(out_path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer outFile.Close()

	if err := tmpl.Execute(outFile, data); err != nil {
		panic(err)
	}

	if err := formatGoFile(out_path); err != nil {
		panic(err)
	}

}

type import_value struct {
	Name     string
	Path     string
	IsIndent bool
}

func (this *import_value) String() string {
	return fmt.Sprintf("Name:%s,Path:%s,IsIndent:%v", this.Name, this.Path, this.IsIndent)
}

type Data struct {
	Importor_all   map[string]*import_value
	Importor_need  map[string]*import_value
	Importor_extra []*import_value
	PackageName    string
	ReqAppend      string
	Client         string
	Server         string
	Module         string
	Funcs          []*func_decl
}

func (this *Data) FixImportorNeedList() {
	for _, f := range this.Funcs {
		for _, in := range f.In {
			if in.ImportPre == "" {
				continue
			}
			if iv, ok := this.Importor_all[in.ImportPre]; ok {
				this.Importor_need[in.ImportPre] = iv
			}
		}
		for _, out := range f.Out {
			if out.ImportPre == "" {
				continue
			}
			if iv, ok := this.Importor_all[out.ImportPre]; ok {
				this.Importor_need[out.ImportPre] = iv
			}
		}
	}
	if imports != "" {
		parts := strings.Split(imports, ",")
		for _, part := range parts {
			subparts := strings.SplitN(part, ":", 2)
			if len(subparts) == 2 {
				name := subparts[0]
				path := subparts[1]
				this.Importor_extra = append(this.Importor_extra, &import_value{
					Name:     name,
					Path:     path,
					IsIndent: name != "",
				})
			} else {
				this.Importor_extra = append(this.Importor_extra, &import_value{
					Name:     "",
					Path:     part,
					IsIndent: false,
				})
			}
		}
	}

}

var data = &Data{
	Importor_all:  map[string]*import_value{},
	Importor_need: map[string]*import_value{},
}

func handleImportSpec(importSpec *ast.ImportSpec) *import_value {
	path := strings.Trim(importSpec.Path.Value, `"`)
	name := filepath.Base(path)
	isIndent := false
	if importSpec.Name != nil {
		name = importSpec.Name.Name
		isIndent = true
	}
	return &import_value{
		Name:     name,
		Path:     path,
		IsIndent: isIndent,
	}
}

type func_decl struct {
	Name               string            //函数名
	NameFunc           string            //函数名，可以被覆盖
	Receiver           string            //接收者类型
	Comments_anotation map[string]string //注解
	Comments_content   []string          //直接替换函数内容
	In                 []*func_param_in_or_out
	Out                []*func_param_in_or_out
	Client             string
	Server             string
	Module             string
	ReqAppend          string
	ReqArg             string //第一个参数名
}

type func_param_in_or_out struct {
	Name            string
	ImportPre       string
	Type            string
	Type_is_pointer bool   //不是指针类型的，用作生成代码时声明
	Type_elem       string //不是指针类型的，用作生成代码时声明
}

func (this *func_param_in_or_out) fixFullTypeName() {
	if this.Type == "" {
		return
	}
	parts := strings.Split(this.Type, ".")
	if len(parts) == 2 {
		this.ImportPre = parts[0]
	}
	this.Type_is_pointer = strings.HasPrefix(this.Type, "*")
	this.Type_elem = strings.TrimPrefix(this.Type, "*")
}

func (this *func_param_in_or_out) String() string {
	return fmt.Sprintf("Name:%s,import_pre:%s,Type:%s", this.Name, this.ImportPre, this.Type)
}

var get_comment_key_value_reg = regexp.MustCompile(`^//\s*@crpc:\s*([^\s]+):\s*([\.\w\s,]*)`)
var get_comment_content_reg = regexp.MustCompile(`^//\s*@crpc_content:\s*([^\s]+)\s*$`)
var get_req_arg_type_reg = regexp.MustCompile(`^([^\s]+)\s+([^\s]+)$`)

func get_req_arg_type(str string) (arg, type_str string, ok bool) {
	matches := get_req_arg_type_reg.FindStringSubmatch(str)
	if len(matches) == 3 {
		return matches[1], matches[2], true
	}
	return "", "", false
}

func get_comment_key_value(comment string) (string, string, bool) {
	matches := get_comment_key_value_reg.FindStringSubmatch(comment)
	if len(matches) == 3 {
		return matches[1], matches[2], true
	}
	return "", "", false
}
func get_comment_content(comment string) (string, bool) {
	matches := get_comment_content_reg.FindStringSubmatch(comment)
	if len(matches) == 2 {
		return matches[1], true
	}
	return "", false
}

func handleFuncDecl(funcSpec *ast.FuncDecl) (res *func_decl) {

	if funcSpec.Type.Params == nil {
		return
	}

	var param_length int
	for _, param := range funcSpec.Type.Params.List {
		for range param.Names {
			param_length++
		}
	}
	if param_length > 2 || param_length < 1 {
		return
	}
	if funcSpec.Type.Results == nil {
		return
	}
	var res_length int
	for _, param := range funcSpec.Type.Results.List {
		if len(param.Names) == 0 {
			res_length++
			continue
		}
		for range param.Names {
			res_length++
		}
	}

	if res_length > 2 || res_length < 1 {
		return
	}

	res_len := len(funcSpec.Type.Results.List)
	res_last_type := getFieldType(funcSpec.Type.Results.List[res_len-1].Type)
	if res_last_type != "error" {
		return
	}

	res = &func_decl{
		Name:               funcSpec.Name.Name,
		Comments_anotation: map[string]string{},
		Comments_content:   []string{},
	}
	// 打印函数的注释
	if funcSpec.Doc != nil {
		for _, comment := range funcSpec.Doc.List {
			if key, value, ok := get_comment_key_value(comment.Text); ok {
				res.Comments_anotation[key] = value
			}
			if content, ok := get_comment_content(comment.Text); ok {
				res.Comments_content = append(res.Comments_content, content)
			}
		}
	}

	if v, ok := res.Comments_anotation["NameFunc"]; ok {
		res.NameFunc = v
	} else {
		res.NameFunc = funcSpec.Name.Name
	}
	if v, ok := res.Comments_anotation["Client"]; ok {
		res.Client = v
	} else {
		res.Client = data.Client
	}
	if v, ok := res.Comments_anotation["Server"]; ok {
		res.Server = v
	} else {
		res.Server = data.Server
	}
	if v, ok := res.Comments_anotation["Module"]; ok {
		res.Module = v
	} else {
		res.Module = data.Module
	}
	if v, ok := res.Comments_anotation["ReqAppend"]; ok {
		res.ReqAppend = v
	} else {
		res.ReqAppend = data.ReqAppend
	}
	// 获取返回值信息
	for index, result := range funcSpec.Type.Results.List {
		res_type := getFieldType(result.Type)
		for _, name := range result.Names {
			v := &func_param_in_or_out{
				Name: name.Name,
				Type: res_type,
			}
			v.fixFullTypeName()
			res.Out = append(res.Out, v)
		}
		// 如果没有命名返回值，打印类型
		if len(result.Names) == 0 {
			var name string
			if res_type == "error" {
				name = "err"
			} else {
				if index == 0 {
					name = "ret"
				} else {
					name = fmt.Sprintf("ret%d", index)
				}
			}
			fmt.Println("param.Type:", name, "type:", res_type, "init_type:", getInitialization(result.Type))

			v := &func_param_in_or_out{
				Name: name,
				Type: res_type,
			}
			v.fixFullTypeName()
			res.Out = append(res.Out, v)
		}
	}

	// 获取参数信息
	if funcSpec.Type.Params != nil {
		param_index := 0
		for _, param := range funcSpec.Type.Params.List {
			for _, name := range param.Names {
				//skip meta.Admin meta.Msg
				if param_length == 2 && param_index == 0 {
					param_index++
					continue
				}
				v := &func_param_in_or_out{
					Name: name.Name,
					Type: getFieldType(param.Type),
				}
				if res.ReqArg == "" {
					res.ReqArg = name.Name
				}
				v.fixFullTypeName()
				res.In = append(res.In, v)
				param_index++
			}
		}
	}

	if req_append := res.ReqAppend; req_append != "" {
		parts := strings.Split(req_append, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if arg, type_str, ok := get_req_arg_type(part); ok {
				v := &func_param_in_or_out{
					Name: arg,
					Type: type_str,
				}
				v.fixFullTypeName()
				res.In = append(res.In, v)
			}
		}
	}
	return
}

// 获取字段类型的字符串表示
func getFieldType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident: // 基本类型或自定义类型
		return t.Name
	case *ast.SelectorExpr: // 包名.类型
		return fmt.Sprintf("%s.%s", t.X, t.Sel.Name)
	case *ast.ArrayType: // 数组类型
		return fmt.Sprintf("[]%s", getFieldType(t.Elt))
	case *ast.MapType: // map 类型
		return fmt.Sprintf("map[%s]%s", getFieldType(t.Key), getFieldType(t.Value))
	case *ast.StarExpr: // 指针类型
		return fmt.Sprintf("*%s", getFieldType(t.X))
	default:
		return fmt.Sprintf("%T", expr) // 其他类型
	}
}

// 获取初始化方式
func getInitialization(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident: // 基本类型或自定义类型
		return fmt.Sprintf("%s{}", t.Name) // 假设使用字面量初始化
	case *ast.ArrayType: // 数组类型
		return fmt.Sprintf("{}") // 使用字面量初始化
	case *ast.StructType: // 数组类型
		return fmt.Sprintf("{}") // 使用字面量初始化
	case *ast.MapType: // map 类型
		return fmt.Sprintf("make(map[%s]%s)", getFieldType(t.Key), getFieldType(t.Value))
	case *ast.StarExpr: // 指针类型
		return fmt.Sprintf("new(%s)", getFieldType(t.X)) // 使用new初始化
	default:
		return fmt.Sprintf("var %s", getFieldType(expr)) // 默认声明
	}
}

//go:embed func_template.go.tmpl
var tmpl string
