package main

import (
	"bytes"
	"embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"strings"
	"unicode"

	"github.com/fatih/structtag"
	"github.com/go-logr/logr"
)

func main() {
	CodegenTypes()
}

var (
	//go:embed *.go.tmpl
	types embed.FS

	tfGenConfigs = []tfGenConfig{
		{
			modelTemplateName: "argocd.go.tmpl",
			sourceFileName:    "./akp/apis/v1alpha1/argocdinstance_types.go",
			structsToIgnore: map[string]bool{
				"ArgoCD":     true,
				"ArgoCDList": true,
			},
		},
		{
			modelTemplateName: "cluster.go.tmpl",
			sourceFileName:    "./akp/apis/v1alpha1/cluster_types.go",
			structsToIgnore: map[string]bool{
				"Cluster":     true,
				"ClusterList": true,
			},
		},
	}

	log logr.Logger
)

type tfGenConfig struct {
	modelTemplateName string
	sourceFileName    string
	structsToIgnore   map[string]bool
	ignoredFieldNames map[string]bool
}

func checkErr(err error) {
	if err != nil {
		_, _ = os.Stderr.Write([]byte(err.Error()))
		os.Exit(-1)
	}
}

func init() {
	var err error
	checkErr(err)
}

func CodegenTypes() {
	for _, c := range tfGenConfigs {
		typesFileOut := c.codegenStructs()
		destFile := "./akp/types/" + strings.TrimSuffix(c.modelTemplateName, ".tmpl")
		err := os.WriteFile(destFile, typesFileOut.Bytes(), 0600)
		checkErr(err)
	}
}

// codegenStructs copies structs from the source file auto-generated file to a new terraform model.go
func (t *tfGenConfig) codegenStructs() *bytes.Buffer {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, t.sourceFileName, nil, 0)
	checkErr(err)

	fsetOut := token.NewFileSet()
	buf := new(bytes.Buffer)

	tmplBytes, err := types.ReadFile(t.modelTemplateName)
	checkErr(err)
	fmt.Fprint(buf, "// This is an auto-generated file. DO NOT EDIT\n")
	fmt.Fprint(buf, string(tmplBytes))
	fmt.Fprint(buf, "\n")

	ast.Inspect(file, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok || typeSpec.Type == nil {
			return true
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		structName := typeSpec.Name.String()
		if _, ok := t.structsToIgnore[structName]; ok {
			log.Info("Skipping generating struct", "struct", structName)
			return true
		}
		log.Info("Code generating struct", "struct", structName)

		var newFieldsList []*ast.Field
		for _, field := range structType.Fields.List {
			if len(field.Names) == 0 {
				continue
			}

			if field.Tag == nil {
				continue
			}
			fieldName := getFieldName(field)
			checkErr(addTerraformTag(field))
			field.Type = convertTerraformType(fieldName, field.Type)
			if objType := rawExtensionToStruct(fieldName, field); objType != nil {
				field.Type = objType
			}
			newFieldsList = append(newFieldsList, field)
		}
		structType.Fields.List = newFieldsList
		_, err = fmt.Fprint(buf, "type ")
		checkErr(err)
		err = printer.Fprint(buf, fsetOut, n)
		checkErr(err)
		_, err = fmt.Fprint(buf, "\n\n")
		checkErr(err)
		return true
	})
	return buf
}

func convertTerraformType(fieldName string, exp ast.Expr) ast.Expr {
	log := log.WithValues("name", fieldName)
	switch f := exp.(type) {
	case *ast.Ident:
		log.Info("converting ident", "ident", f)
		if attributeType := getTerraformAttributeType(f.Name); attributeType != nil {
			return attributeType
		}
	case *ast.ArrayType:
		log.Info("converting array", "array", f)
		elt, _ := resolvePointer(f.Elt)
		arrayTypeName := fmt.Sprintf("%s", elt)
		if attributeType := getTerraformAttributeType(arrayTypeName); attributeType != nil {
			f.Elt = attributeType
			return f
		}
	case *ast.MapType:
		log.Info("converting map", "map", f)
		panic("map type is not yet supported")
	case *ast.StarExpr:
		log.Info("converting pointer", "pointer", f)
		ptr, ok := resolvePointer(exp)
		typeName := fmt.Sprintf("%s", ptr)
		if attributeType := getTerraformAttributeType(typeName); attributeType != nil {
			return attributeType
		}
		if ok {
			f.X = convertTerraformType(fieldName, ptr)
			return f
		}
		return convertTerraformType(fieldName, ptr)
	}
	return exp
}

// getTerraformAttributeType gets the terraform framework attribute type from the go primitive types.
// https://developer.hashicorp.com/terraform/plugin/framework/handling-data/attributes#framework-attribute-types
func getTerraformAttributeType(typeName string) ast.Expr {
	switch typeName {
	case "string":
		return &ast.SelectorExpr{
			X:   &ast.Ident{Name: "types"},
			Sel: &ast.Ident{Name: "String"},
		}
	case "bool":
		return &ast.SelectorExpr{
			X:   &ast.Ident{Name: "types"},
			Sel: &ast.Ident{Name: "Bool"},
		}
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return &ast.SelectorExpr{
			X:   &ast.Ident{Name: "types"},
			Sel: &ast.Ident{Name: "Int64"},
		}
	case "float32", "float64":
		return &ast.SelectorExpr{
			X:   &ast.Ident{Name: "types"},
			Sel: &ast.Ident{Name: "Float64"},
		}
	case "ClusterSize":
		return &ast.SelectorExpr{
			X:   &ast.Ident{Name: "types"},
			Sel: &ast.Ident{Name: "String"},
		}
	case "ClusterCustomization":
		return &ast.SelectorExpr{
			X:   &ast.Ident{Name: "types"},
			Sel: &ast.Ident{Name: "Object"},
		}
	}
	return nil
}

func rawExtensionToStruct(fieldName string, field *ast.Field) ast.Expr {
	if fieldName != "Kustomization" {
		return nil
	}
	expr, _ := resolvePointer(field.Type)
	f, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return nil
	}
	ident := f.X.(*ast.Ident)
	typeName := fmt.Sprintf("%s.%s", ident.Name, f.Sel.Name)
	if typeName == "runtime.RawExtension" {
		ident.Name = "types"
		f.Sel.Name = "String"
	} else {
		panic(fmt.Sprintf("I dont know how to convert for field %s of type %s", fieldName, typeName))
	}
	return expr
}

func addTerraformTag(field *ast.Field) error {
	tagValue := strings.TrimPrefix(field.Tag.Value, "`")
	tagValue = strings.TrimSuffix(tagValue, "`")
	tags, err := structtag.Parse(tagValue)
	jsonTag, err := tags.Get("json")
	if err != nil {
		return err
	}
	snakeCaseName := camelToSnakeCase(jsonTag.Name)
	err = tags.Set(&structtag.Tag{
		Key:  "tfsdk",
		Name: snakeCaseName,
	})
	if err != nil {
		return err
	}
	field.Tag.Value = fmt.Sprintf("`%s`", tags.String())
	return nil
}

func camelToSnakeCase(input string) string {
	var buf bytes.Buffer

	for i, r := range input {
		if unicode.IsUpper(r) {
			if i > 0 {
				buf.WriteByte('_')
			}
			buf.WriteRune(unicode.ToLower(r))
		} else {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

// resolvePointer returns the underlying type of a field
func resolvePointer(exp ast.Expr) (ast.Expr, bool) {
	ptr, ok := exp.(*ast.StarExpr)
	if !ok {
		return exp, false
	}
	return ptr.X, true
}

// getFieldName is a helper to return the field name of a struct
func getFieldName(field *ast.Field) string {
	if len(field.Names) != 1 {
		panic(fmt.Sprintf("  found field with multiple names %s", field.Names))
	}
	return field.Names[0].String()
}
