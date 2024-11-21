package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/samber/lo"
)

var inc_type_keys = []string{
	"int",
	"int8",
	"int16",
	"int32",
	"int64",
	"uint",
	"uint8",
	"uint16",
	"uint32",
	"uint64",
	"float32",
	"float64",
}

type Field struct {
	StructName string //构建模版的时候无法拿到上层的东西,with不工作
	Name       string
	IsInc      bool
	Type       string
}

type StructData struct {
	StructName string
	Fields     []Field
}

type FileStructData struct {
	PackageName string
	StructDatas []StructData
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

func main() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	filename := os.Getenv("GOFILE")
	if filename == "" {
		return
	}
	if !strings.HasSuffix(filename, ".go") {
		return
	}
	file := filepath.Join(dir, filename)
	filename = filename[:len(filename)-3]
	filename_new := fmt.Sprintf("%s_struct_gen.go", filename)
	filepath_new := filepath.Join(dir, filename_new)

	// 解析 Go 文件
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 获取包名
	packageName := node.Name.Name
	fileStructData := &FileStructData{
		PackageName: packageName,
	}

	var structs []StructData
	// 遍历 AST
	ast.Inspect(node, func(n ast.Node) bool {
		// 查找结构体类型
		typeSpec, ok := n.(*ast.TypeSpec)
		if ok {
			structType, ok := typeSpec.Type.(*ast.StructType)
			if ok {
				struct_name := typeSpec.Name.Name
				fields := []Field{}

				for _, field := range structType.Fields.List {
					for _, name := range field.Names {
						field := Field{
							StructName: struct_name,
							Name:       name.Name,
							Type:       getFieldType(field.Type),
						}
						field.IsInc = lo.Contains(inc_type_keys, field.Type)
						fields = append(fields, field)
					}
				}
				structs = append(structs, StructData{
					StructName: struct_name,
					Fields:     fields,
				})
			}
		}
		return true
	})
	fileStructData.StructDatas = structs

	// tmpl, err := template.ParseFiles("struct_template.go.tmpl")
	tmpl, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	outFile, err := os.OpenFile(filepath_new, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer outFile.Close()

	if err := tmpl.Execute(outFile, fileStructData); err != nil {
		panic(err)
	}

	// for _, s := range structs {
	// 	fmt.Printf("Struct: %s\n", s.StructName)
	// 	for _, field := range s.Fields {
	// 		fmt.Printf("  Field: %s, Type: %s,Inc:%v\n", field.Name, field.Type, field.IsInc)
	// 	}
	// }

}

const tmpl = `package {{.PackageName}}
{{- range .StructDatas}}

// Set{{.StructName}} 指定的值,需要指定keys
func (this *{{.StructName}}) Set(delta *{{.StructName}}, keys ...string) {
	if delta == nil {
		return
	}
	for _, key := range keys {
		switch key {
        {{- range .Fields }}
		case "{{.Name}}":
			this.{{.Name}} = delta.{{.Name}}
         {{- end }}
		}
	}
}

// Inc{{.StructName}} 指定的值,需要指定keys
func (this *{{.StructName}}) Inc(delta *{{.StructName}}, keys ...string) {
	if delta == nil {
		return
	}
	for _, key := range keys {
		switch key {
        {{- range .Fields }}
        {{- if .IsInc  }}
		case "{{.Name}}":
			this.{{.Name}} += delta.{{.Name}}
        {{- end }}
        {{- end }}
		}
	}
}

// Add{{.StructName}} 指定的值,只支持数值类型
func (this *{{.StructName}}) Add(delta *{{.StructName}}) {
	if delta == nil {
		return
	}
    {{- range .Fields }}
    {{- if .IsInc  }}
	this.{{.Name}} += delta.{{.Name}}
    {{- end }}
    {{- end }}
}

// 定义{{.StructName}} 对应字段的key
{{- range .Fields }}
var {{ .StructName}}_{{.Name}} = "{{ .Name}}"
{{- end }}


{{- end }}
`
