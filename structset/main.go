//go:generate structset -k false
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
	"reflect"
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

func getFieldTag(field *ast.Field) string {
	if field.Tag != nil {
		tag := strings.Trim(field.Tag.Value, "`")
		return tag
	}
	return ""
}

var tag_name = "structset"

func shouldIgnoreField(tag string) bool {
	// 根据 structset 标签判断是否忽略字段
	tag_value, _ := getTagValue(tag, tag_name)
	if tag_value == "-" {
		return true
	}
	return false
}

func getTagValue(tag, tag_name string) (tag_value string, qualifier []string) {
	structTag := reflect.StructTag(tag)
	tag_value = structTag.Get(tag_name)
	tag_values := strings.Split(tag_value, ",")
	tag_value, is := lo.First(tag_values)
	if is {
		qualifier = tag_values[1:]
	}
	return
}

func fix_default_tag_value(tag_value, default_value string) string {
	if tag_value == "" {
		tag_value = default_value
	}
	return tag_value
}

// 格式化 Go 文件
func formatGoFile(filePath string) error {
	cmd := exec.Command("gofmt", "-w", filePath)
	return cmd.Run()
}

func main() {
	flag.String("useage", "useage", "structset -k=false -a=false -i=false --um=false -s=false --im=false --sub=ccc --tagname=structset")
	flag.StringVar(&file_path, "f", "", "需要解析的文件")
	flag.StringVar(&out_path, "o", "", "输出的新文件")
	flag.StringVar(&suffix, "sub", "_struct_gen", "输出的新文件的后缀")
	flag.StringVar(&tag_name, "tagname", "structset", "提取的tag名称.eg:json")
	flag.BoolVar(&fileStructData.IsKey, "k", true, "是否生成结构体的keys常量")
	flag.BoolVar(&fileStructData.IsJsonM, "jm", true, "是否生成结构体的bson.M")
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
					tag := getFieldTag(field)
					if shouldIgnoreField(tag) {
						continue
					}
					tag_value, _ := getTagValue(tag, tag_name)
					json_tag_value, _ := getTagValue(tag, "json")
					bson_tag_value, _ := getTagValue(tag, "bson")
					// if idx := strings.Index(tag_name_value, ","); idx != -1 {
					// 	tag_name_value = tag_name_value[:idx] // 只保留第一个逗号前的部分
					// }
					// if tag_value == "" {
					// 	continue
					// }
					for _, name := range field.Names {
						filedType := getFieldType(field.Type)
						if tag_value == "" {
							tag_value = name.Name // 默认使用字段名作为标签值
						}
						if json_tag_value == "" {
							json_tag_value = name.Name // 默认使用字段名作为标签值
						}
						Name := name.Name
						field := Field{
							StructName:   struct_name,
							Name:         Name,
							TagValue:     fix_default_tag_value(tag_value, Name),
							JsonTagValue: fix_default_tag_value(json_tag_value, Name),
							BsonTagValue: fix_default_tag_value(bson_tag_value, Name),
							Type:         filedType,
							IsInc:        lo.Contains(inc_type_keys, filedType),
						}
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

//go:embed struct_template.go.tmpl
var tmpl string
