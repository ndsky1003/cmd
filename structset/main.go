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

var (
	// Version is set by build flags
	Version = "dev"
)

const (
	StructFieldAttr_inc     = "inc"
	StructFieldAttr_no_inc  = "noinc"
	StructFieldAttr_add     = "add"
	StructFieldAttr_no_add  = "noadd"
	StructFieldAttr_copy    = "copy"
	StructFieldAttr_no_copy = "nocopy"
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

func getTagValue(tag, tag_name string) (tag_value string, qualifier map[string]string) {
	structTag := reflect.StructTag(tag)
	tag_value = structTag.Get(tag_name)
	tag_values := strings.Split(tag_value, ",")
	tag_value, is := lo.First(tag_values)
	qualifier = make(map[string]string)
	if is {
		qualifier_arr := tag_values[1:]
		qualifier = lo.SliceToMap(qualifier_arr, func(item string) (string, string) {
			parts := strings.SplitN(item, ":", 2)
			if len(parts) == 2 {
				return parts[0], parts[1]
			}
			return parts[0], ""
		})
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
	cmd := exec.Command("goimports", "-w", filePath)
	return cmd.Run()
}

func main() {
	flag.String("useage", "useage", "structset -k=false  --sub=ccc --tagname=structset")
	attr := flag.Bool("attr", false, "显示可以可设置的属性")
	flag.StringVar(&file_path, "f", "", "需要解析的文件")
	flag.StringVar(&out_path, "o", "", "输出的新文件")
	flag.StringVar(&suffix, "sub", "_struct_gen", "输出的新文件的后缀")
	flag.StringVar(&tag_name, "tagname", "structset", "提取的tag名称.eg:json")
	flag.BoolVar(&fileStructData.IsKey, "k", true, "是否生成结构体的keys常量")
	flag.BoolVar(&fileStructData.IsJsonM, "jm", true, "是否生成结构体的bson.M")
	flag.BoolVar(&fileStructData.IsBsonM, "bm", true, "是否生成结构体的bson.M,$set")
	flag.BoolVar(&fileStructData.IsIncM, "im", true, "是否生成结构体的bson.M,$inc")
	flag.BoolVar(&fileStructData.IsBsonD, "bd", true, "是否生成结构体的bson.D,$set")
	flag.BoolVar(&fileStructData.IsIncD, "id", true, "是否生成结构体的bson.D,$inc")
	flag.BoolVar(&fileStructData.IsSet, "s", true, "是否生成结构体的Set(detal *StructData,keys []string)")
	flag.BoolVar(&fileStructData.IsInc, "i", true, "是否生成结构体的Inc(detal *StructData,keys []string)")
	flag.BoolVar(&fileStructData.IsAdd, "a", true, "是否生成结构体的Add(detal *StructData)")
	flag.BoolVar(&fileStructData.IsCopy, "c", true, "是否生成结构体的Copy(keys []string)")
	v := flag.Bool("v", false, "print version information and exit")
	flag.BoolVar(v, "version", false, "same as -v")
	flag.Parse()
	if *v {
		fmt.Println(Version)
		os.Exit(0)
	}
	if *attr {
		fmt.Println("attr:", []string{
			StructFieldAttr_inc,
			StructFieldAttr_no_inc,
			StructFieldAttr_add,
			StructFieldAttr_no_add,
			StructFieldAttr_copy,
			StructFieldAttr_no_copy})
		fmt.Printf("可自由定制方法或者函数，函数以f_开头即可，其他都是方法\n")
		fmt.Println("usage:structset:\"ccc,copy:f_cc,add:add_method\"")
		return
	}

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

				var is_have_inc_key = false
				var is_have_add_key = false
				var is_have_copy_key = false
				for _, field := range structType.Fields.List {
					tag := getFieldTag(field)
					if shouldIgnoreField(tag) {
						continue
					}
					tag_value, qualifiers := getTagValue(tag, tag_name)
					json_tag_value, _ := getTagValue(tag, "json")
					bson_tag_value, _ := getTagValue(tag, "bson")
					filedType := getFieldType(field.Type)

					is_inc := lo.Contains(inc_type_keys, filedType)
					is_inc, _, _ = get_tag_attr_is_method_func(qualifiers, is_inc, StructFieldAttr_inc, StructFieldAttr_no_inc)

					is_add := lo.Contains(inc_type_keys, filedType)
					is_add, add_f, add_m := get_tag_attr_is_method_func(qualifiers, is_add, StructFieldAttr_add, StructFieldAttr_no_add)

					is_copy := true
					is_copy, copy_f, copy_m := get_tag_attr_is_method_func(qualifiers, is_copy, StructFieldAttr_copy, StructFieldAttr_no_copy)
					for _, name := range field.Names {
						Name := name.Name
						field := Field{
							StructName:   struct_name,
							Name:         Name,
							TagValue:     fix_default_tag_value(tag_value, Name),
							JsonTagValue: fix_default_tag_value(json_tag_value, Name),
							BsonTagValue: fix_default_tag_value(bson_tag_value, Name),
							Type:         filedType,
							IsInc:        is_inc,
							IsAdd:        is_add,
							AddFunc:      add_f,
							AddMethod:    add_m,
							IsCopy:       is_copy,
							CopyFunc:     copy_f,
							CopyMethod:   copy_m,
						}
						if field.IsInc {
							is_have_inc_key = true
						}
						if field.IsAdd {
							is_have_add_key = true
						}
						if field.IsCopy {
							is_have_copy_key = true
						}
						fields = append(fields, field)
					}
				}
				structs = append(structs, StructData{
					StructName:    struct_name,
					IsHaveIncKey:  is_have_inc_key,
					IsHaveAddKey:  is_have_add_key,
					IsHaveCopyKey: is_have_copy_key,
					Fields:        fields,
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
	IsAdd        bool
	IsCopy       bool
	AddMethod    string
	CopyMethod   string //m 是方法，属于结构体。f_是函数，属于包
	AddFunc      string
	CopyFunc     string //m 是方法，属于结构体。f_是函数，属于包
}

type StructData struct {
	StructName    string
	IsHaveIncKey  bool
	IsHaveAddKey  bool
	IsHaveCopyKey bool
	Fields        []Field
}

func get_tag_attr_is_method_func(attrs map[string]string, default_attr_value bool, attr, no_attr string) (is bool, func_, method string) {
	is = default_attr_value

	if _, ok := attrs[no_attr]; ok {
		is = false
		return
	}

	if m_or_f, ok := attrs[attr]; ok {
		if m_or_f != "" {
			if strings.HasPrefix(m_or_f, "f_") {
				func_ = strings.TrimLeft(m_or_f, "f_")
			} else {
				if strings.HasPrefix(m_or_f, "m_") {
					method = strings.TrimLeft(m_or_f, "m_")
				} else {
					method = m_or_f
				}
			}
		}
		is = true
	}
	return
}

type FileStructData struct {
	PackageName string
	StructDatas []StructData
	IsKey       bool
	IsJsonM     bool
	IsBsonM     bool //bson.M
	IsIncM      bool //bson.M
	IsBsonD     bool //bson.D
	IsIncD      bool //bson.D
	IsSet       bool
	IsInc       bool
	IsAdd       bool
	IsCopy      bool
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
