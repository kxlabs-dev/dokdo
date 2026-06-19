package builder

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type FieldInfo struct {
	Name        string
	TypeStr     string
	Nullable    bool
	IsAnonSlice bool
	AnonFields  []FieldInfo
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

func extractAnonStructFields(st *ast.StructType) []FieldInfo {
	var fields []FieldInfo
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue
		}
		typeStr := exprToTypeStr(f.Type)
		for _, name := range f.Names {
			if !ast.IsExported(name.Name) {
				continue
			}
			fields = append(fields, FieldInfo{
				Name:     name.Name,
				TypeStr:  typeStr,
				Nullable: strings.HasPrefix(typeStr, "*"),
			})
		}
	}
	return fields
}

func ParseGoFiles(dir string) (map[string]*TypeInfo, error) {
	result := make(map[string]*TypeInfo)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return result, err
	}

	fset := token.NewFileSet()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		f, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, 0)
		if err != nil {
			return nil, err
		}

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
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				if !ast.IsExported(typeSpec.Name.Name) {
					continue
				}

				info := &TypeInfo{Name: typeSpec.Name.Name}
				for _, field := range structType.Fields.List {
					if len(field.Names) == 0 {
						continue
					}

					if arrType, ok := field.Type.(*ast.ArrayType); ok {
						if anonSt, ok := arrType.Elt.(*ast.StructType); ok {
							anonFields := extractAnonStructFields(anonSt)
							for _, name := range field.Names {
								if !ast.IsExported(name.Name) {
									continue
								}
								info.Fields = append(info.Fields, FieldInfo{
									Name:        name.Name,
									IsAnonSlice: true,
									AnonFields:  anonFields,
								})
							}
							continue
						}
						if ident, ok := arrType.Elt.(*ast.Ident); ok && !goPrimitives[ident.Name] {
							for _, name := range field.Names {
								if ast.IsExported(name.Name) {
									return nil, fmt.Errorf(
										"field '%s' uses unsupported type '[]%s'. Use anonymous struct slice '[]struct{...}' instead.",
										name.Name, ident.Name,
									)
								}
							}
							continue
						}
					} else if _, ok := field.Type.(*ast.MapType); ok {
						for _, name := range field.Names {
							if ast.IsExported(name.Name) {
								return nil, fmt.Errorf(
									"field '%s' uses unsupported type 'map'. Use anonymous struct slice '[]struct{...}' instead.",
									name.Name,
								)
							}
						}
						continue
					}

					typeStr := exprToTypeStr(field.Type)
					for _, name := range field.Names {
						if !ast.IsExported(name.Name) {
							continue
						}
						info.Fields = append(info.Fields, FieldInfo{
							Name:     name.Name,
							TypeStr:  typeStr,
							Nullable: strings.HasPrefix(typeStr, "*"),
						})
					}
				}
				result[info.Name] = info
			}
		}
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
