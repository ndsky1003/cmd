//go:generate structset -k false
package main

import (
	"flag"
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

var (
	file_path string
	out_path  string
	suffix    string
)

var fileStructData = &FileStructData{}

func init() {
}

func main() {
	flag.String("useage", "useage", "structset -k=false -a=false -i=false --um=false -s=false --im=false --sub=ccc")
	flag.StringVar(&file_path, "f", "", "需要解析的文件")
	flag.StringVar(&out_path, "o", "", "输出的新文件")
	flag.StringVar(&suffix, "sub", "_struct_gen", "输出的新文件的后缀")
	flag.BoolVar(&fileStructData.IsKey, "k", true, "是否生成结构体的keys常量")
	flag.BoolVar(&fileStructData.IsUpM, "um", true, "是否生成结构体的bson.D,$set")
	flag.BoolVar(&fileStructData.IsIncM, "im", true, "是否生成结构体的bson.D,$inc")
	flag.BoolVar(&fileStructData.IsSet, "s", true, "是否生成结构体的Set(detal *StructData,keys []string)")
	flag.BoolVar(&fileStructData.IsInc, "i", true, "是否生成结构体的Inc(detal *StructData,keys []string)")
	flag.BoolVar(&fileStructData.IsAdd, "a", true, "是否生成结构体的Add(detal *StructData)")
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
		typeSpec, ok := n.(*ast.TypeSpec)
		if ok {
			structType, ok := typeSpec.Type.(*ast.StructType)
			if ok {
				struct_name := typeSpec.Name.Name
				fields := []Field{}

				var is_have_number = false
				for _, field := range structType.Fields.List {
					for _, name := range field.Names {
						field := Field{
							StructName: struct_name,
							Name:       name.Name,
							Type:       getFieldType(field.Type),
						}
						field.IsInc = lo.Contains(inc_type_keys, field.Type)
						if field.IsInc {
							is_have_number = true
						}
						fields = append(fields, field)
					}
				}
				structs = append(structs, StructData{
					StructName:      struct_name,
					IsHashNumberKey: is_have_number,
					Fields:          fields,
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

	outFile, err := os.OpenFile(out_path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
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

type Field struct {
	StructName string //构建模版的时候无法拿到上层的东西,with不工作
	Name       string
	IsInc      bool
	Type       string
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

const tmpl = `package {{.PackageName}}

import (
	"errors"

{{- if or $.IsUpM $.IsIncM}}
	"go.mongodb.org/mongo-driver/bson"
{{- end }}
)

{{- range .StructDatas}}
{{- if $.IsKey}}
// 定义{{.StructName}} 对应字段的key
{{- range .Fields }}
const {{ .StructName}}_{{.Name}} = "{{ .Name}}"
{{- end }}
{{- end }}


{{- if $.IsUpM}}
// Set{{.StructName}} 指定的值,upM
func (this *{{.StructName}}) GenUpdateMap(keys []string) ([]bson.E, error) {
    if this == nil {
        return nil, errors.New("receiver is nil")
    }
    if len(keys) == 0 {
        return nil, errors.New("no keys provided")
    }
	upM := make([]bson.E, 0, len(keys))
	for _, key := range keys {
		switch key {
        {{- range .Fields }}
		case "{{.Name}}":
		    upM = append(upM, bson.E{Key: key, Value: this.{{.Name}}})
        {{- end }}
		}
	}
    return upM, nil
}
{{- end }}

{{- if $.IsIncM}}
// Set{{.StructName}} 指定的值,incM
func (this *{{.StructName}}) GenIncM(keys []string) ([]bson.E, error) {
    if this == nil {
        return nil, errors.New("receiver is nil")
    }
    if len(keys) == 0 {
        return nil, errors.New("no keys provided")
    }
    {{- if .IsHashNumberKey}}
	upM := make([]bson.E, 0, len(keys))
	for _, key := range keys {
		switch key {
        {{- range .Fields }}
        {{- if .IsInc  }}
		case "{{.Name}}":
		    upM = append(upM, bson.E{Key: key, Value: this.{{.Name}}})
        {{- end }}
        {{- end }}
		}
	}
    return upM, nil
    {{- else}}
	return nil, errors.New("key is not has number")
	{{- end }}
}
{{- end }}

{{- if $.IsSet}}
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
{{- end }}

{{- if $.IsInc}}
// Inc{{.StructName}} 指定的值,需要指定keys
func (this *{{.StructName}}) Inc(delta *{{.StructName}}, keys ...string) {
	if delta == nil {
		return
	}
    {{- if .IsHashNumberKey}}
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
    {{- end}}
}
{{- end }}

{{- if $.IsAdd}}
// Add{{.StructName}} 指定的值,只支持数值类型
func (this *{{.StructName}}) Add(delta *{{.StructName}}) {
	if delta == nil {
		return
	}
    {{- if .IsHashNumberKey}}
    {{- range .Fields }}
    {{- if .IsInc  }}
	this.{{.Name}} += delta.{{.Name}}
    {{- end }}
    {{- end }}
    {{- end}}
}
{{- end }}

{{- end }}
`
