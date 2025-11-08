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
	"strings"
	"text/template"
)

var (
	file_path string
	out_path  string
	suffix    string
)

var fileStructData = &FileStructData{}

// 格式化 Go 文件
func formatGoFile(filePath string) error {
	cmd := exec.Command("gofmt", "-w", filePath)
	return cmd.Run()
}

func main() {
	flag.String("useage", "useage", "gencrpc --sub=ccc")
	flag.StringVar(&file_path, "f", "", "需要解析的文件")
	flag.StringVar(&out_path, "o", "", "输出的新文件")
	flag.StringVar(&suffix, "sub", "_struct_gen", "输出的新文件的后缀")
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
	fileStructData.PackageName = node.Name.Name

	var structs []StructData
	// 遍历 AST
	ast.Inspect(node, func(n ast.Node) bool {
		// 查找结构体类型
		funcSpec, ok := n.(*ast.FuncDecl)
		if ok {
			if funcSpec.Recv == nil {
				return true
			}
			fmt.Println("name:", funcSpec.Name.Name)
			for _, field := range funcSpec.Recv.List {
				// 打印接收者类型
				fmt.Printf("Receiver: %v\n", getFieldType(field.Type))
			}
			// 打印函数的注释
			if funcSpec.Doc != nil {
				fmt.Println("Comments:")
				for _, comment := range funcSpec.Doc.List {
					fmt.Println(comment.Text)
				}
			}
			for _, v := range funcSpec.Type.Params.List {
				fmt.Println("dd:", v.Names)
			}

			// 获取参数信息
			if funcSpec.Type.Params != nil {
				fmt.Println("Parameters:")
				for _, param := range funcSpec.Type.Params.List {
					for _, name := range param.Names {
						fmt.Printf(" - %v: %v\n", name.Name, getFieldType(param.Type))
					}
				}
			} else {
				fmt.Println("No parameters.")
			}

			// 获取返回值信息
			if funcSpec.Type.Results != nil {
				fmt.Println("Returns:")
				for _, result := range funcSpec.Type.Results.List {
					// 如果有命名返回值，打印名称
					for _, name := range result.Names {
						fmt.Printf(" - %s: %s\n", name.Name, getFieldType(result.Type))
					}
					// 如果没有命名返回值，打印类型
					if len(result.Names) == 0 {
						fmt.Printf(" - Return type: %v\n", getFieldType(result.Type))
					}
				}
			} else {
				fmt.Println("No return values.")
			}
		}
		return true
	})
	return
	fileStructData.StructDatas = structs

	// tmpl, err := template.ParseFiles("struct_template.go.tmpl")
	tmpl, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	outFile, err := os.OpenFile(out_path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer outFile.Close()

	if err := tmpl.Execute(outFile, fileStructData); err != nil {
		panic(err)
	}

	if err := formatGoFile(out_path); err != nil {
		panic(err)
	}

	// for _, s := range structs {
	// 	fmt.Printf("Struct: %s\n", s.StructName)
	// 	for _, field := range s.Fields {
	// 		fmt.Printf("  Field: %s, Type: %s,Inc:%v\n", field.Name, field.Type, field.IsInc)
	// 	}
	// }

}

type Field struct {
	StructName   string //构建模版的时候无法拿到上层的东西,with不工作
	Name         string //结构体的字段名
	TagValue     string //结构体的标签名：默认是structset
	JsonTagValue string //结构体的标签名：json
	BsonTagValue string //结构体的标签名：bson
	Type         string
	Tag          string // 新增字段用于存储标签信息
	IsInc        bool
}

type StructData struct {
	StructName      string
	IsHashNumberKey bool
	Fields          []Field
}

type FileStructData struct {
	PackageName string
	StructDatas []StructData
	IsKey       bool
	IsJsonM     bool
	IsUpM       bool
	IsIncM      bool
	IsSet       bool
	IsInc       bool
	IsAdd       bool
}

// func getStructName(expr ast.Expr) string {
// 	switch t := expr.(type) {
// 	case *ast.Ident: // 基本类型或自定义类型
// 		return t.Name
// 	case *ast.SelectorExpr: // 包名.类型
// 		return fmt.Sprintf("%s.%s", t.X, t.Sel.Name)
// 	case *ast.ArrayType: // 数组类型
// 		return getStructName(t.Elt)
// 	case *ast.MapType: // map 类型
// 		return getStructName(t.Key)
// 	case *ast.StarExpr: // 指针类型
// 		return getStructName(t.X)
// 	default:
// 		return fmt.Sprintf("%T", expr) // 其他类型
// 	}
// }

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

//go:embed func_template.go.tmpl
var tmpl string
