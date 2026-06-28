package builder

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type FieldInfo struct {
	Name           string
	TypeStr        string
	Nullable       bool
	IsNamedSlice   bool
	SliceElemType  string
	SliceFields    []FieldInfo
	IsNamedStruct  bool
	StructElemType string
	StructFields   []FieldInfo
}

type TypeInfo struct {
	Name   string
	Fields []FieldInfo
}

var goPrimitives = map[string]bool{
	"string": true, "bool": true, "byte": true, "rune": true,
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true, "complex64": true, "complex128": true,
	"error": true,
}

func buildFields(st *ast.StructType, localTypes map[string]*ast.StructType, visiting map[string]bool, path []string) ([]FieldInfo, error) {
	var fields []FieldInfo
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue
		}
		for _, nameIdent := range f.Names {
			if !ast.IsExported(nameIdent.Name) {
				continue
			}
			fi, err := resolveExpr(nameIdent.Name, f.Type, false, localTypes, visiting, path)
			if err != nil {
				return nil, err
			}
			fields = append(fields, fi)
		}
	}
	return fields, nil
}

func resolveExpr(name string, expr ast.Expr, nullable bool, localTypes map[string]*ast.StructType, visiting map[string]bool, path []string) (FieldInfo, error) {
	switch e := expr.(type) {
	case *ast.StarExpr:
		if nullable {
			return FieldInfo{}, fmt.Errorf("field '%s': pointer to pointer (**T) is not supported.", name)
		}
		return resolveExpr(name, e.X, true, localTypes, visiting, path)

	case *ast.ArrayType:
		if _, ok := e.Elt.(*ast.StructType); ok {
			return FieldInfo{}, fmt.Errorf(
				"field '%s': anonymous struct slice is not allowed. Use named struct instead.", name)
		}
		ident, ok := e.Elt.(*ast.Ident)
		if !ok {
			return FieldInfo{}, fmt.Errorf("field '%s': unsupported array element type.", name)
		}
		if goPrimitives[ident.Name] {
			return FieldInfo{Name: name, TypeStr: "[]" + ident.Name, Nullable: nullable}, nil
		}
		st, ok := localTypes[ident.Name]
		if !ok {
			return FieldInfo{}, fmt.Errorf(
				"field '%s' uses unsupported type '[]%s'. Use named struct in the same file.",
				name, ident.Name)
		}
		newPath := append(path, ident.Name)
		if visiting[ident.Name] {
			return FieldInfo{}, fmt.Errorf("circular reference detected: %s",
				strings.Join(newPath, " → "))
		}
		visiting[ident.Name] = true
		sliceFields, err := buildFields(st, localTypes, visiting, newPath)
		delete(visiting, ident.Name)
		if err != nil {
			return FieldInfo{}, err
		}
		return FieldInfo{
			Name: name, IsNamedSlice: true,
			SliceElemType: ident.Name, SliceFields: sliceFields, Nullable: nullable,
		}, nil

	case *ast.Ident:
		if goPrimitives[e.Name] {
			return FieldInfo{Name: name, TypeStr: e.Name, Nullable: nullable}, nil
		}
		st, ok := localTypes[e.Name]
		if !ok {
			return FieldInfo{}, fmt.Errorf(
				"field '%s' uses unsupported type '%s'. Use named struct in the same file.",
				name, e.Name)
		}
		newPath := append(path, e.Name)
		if visiting[e.Name] {
			return FieldInfo{}, fmt.Errorf("circular reference detected: %s",
				strings.Join(newPath, " → "))
		}
		visiting[e.Name] = true
		structFields, err := buildFields(st, localTypes, visiting, newPath)
		delete(visiting, e.Name)
		if err != nil {
			return FieldInfo{}, err
		}
		return FieldInfo{
			Name:           name,
			IsNamedStruct:  true,
			StructElemType: e.Name,
			StructFields:   structFields,
			Nullable:       nullable,
		}, nil

	case *ast.MapType:
		return FieldInfo{}, fmt.Errorf("field '%s' uses unsupported type 'map'.", name)

	default:
		return FieldInfo{Name: name, TypeStr: exprToTypeStr(expr), Nullable: nullable}, nil
	}
}

func ParseGoFile(path string) (map[string]*TypeInfo, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}

	// Pass 1: 파일 내 exported struct 타입 수집 (필드 처리 없음)
	localTypes := make(map[string]*ast.StructType)
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			if !ast.IsExported(typeSpec.Name.Name) {
				continue
			}
			localTypes[typeSpec.Name.Name] = st
		}
	}

	// Pass 2: 각 타입의 필드를 재귀 처리
	result := make(map[string]*TypeInfo)
	for typeName, st := range localTypes {
		visiting := map[string]bool{typeName: true}
		path := []string{typeName}
		fields, err := buildFields(st, localTypes, visiting, path)
		if err != nil {
			return nil, err
		}
		result[typeName] = &TypeInfo{Name: typeName, Fields: fields}
	}

	return result, nil
}

func exprToTypeStr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprToTypeStr(e.X)
	case *ast.ArrayType:
		return "[]" + exprToTypeStr(e.Elt)
	case *ast.MapType:
		return "map[" + exprToTypeStr(e.Key) + "]" + exprToTypeStr(e.Value)
	case *ast.StructType:
		return "struct"
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return ""
	}
}
