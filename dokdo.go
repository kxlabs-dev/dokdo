package dokdo

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kxlabs-dev/dokdo/builder"
	"github.com/kxlabs-dev/dokdo/parser"
)

type Dokdo struct {
	queries map[string]*QueryEntry
}

type QueryEntry struct {
	Node     *parser.QueryNode
	TypeInfo *builder.TypeInfo
	File     string
}

func Load(root string) (*Dokdo, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	queries := make(map[string]*QueryEntry)
	typeInfoCache := make(map[string]map[string]*builder.TypeInfo)

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".kx") {
			return nil
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(absPath, absRoot) {
			return &PathTraversalError{Path: path}
		}

		dir := filepath.Dir(absPath)
		if _, ok := typeInfoCache[dir]; !ok {
			typeInfoCache[dir], err = builder.ParseGoFiles(dir)
			if err != nil {
				return &BuildError{Message: err.Error()}
			}
		}
		dirTypes := typeInfoCache[dir]

		data, err := os.ReadFile(absPath)
		if err != nil {
			return &FileNotFoundError{Path: absPath}
		}

		qf, err := parser.ParseFile(string(data), absPath)
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(absRoot, absPath)
		if err != nil {
			return err
		}
		relNoExt := strings.TrimSuffix(rel, filepath.Ext(rel))
		fileNamespace := filepath.ToSlash(relNoExt)

		for i := range qf.Queries {
			q := &qf.Queries[i]
			key := fileNamespace + "#" + q.Name

			var typeInfo *builder.TypeInfo
			if q.ParamRef != "" {
				parts := strings.SplitN(q.ParamRef, "#", 2)
				typeNamePart := parts[len(parts)-1]
				ti, ok := dirTypes[typeNamePart]
				if !ok {
					return &BuildError{Message: "type not found: " + typeNamePart}
				}
				typeInfo = ti
			}

			queries[key] = &QueryEntry{
				Node:     q,
				TypeInfo: typeInfo,
				File:     absPath,
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &Dokdo{queries: queries}, nil
}

func (d *Dokdo) Build(target string, params interface{}) (string, []interface{}, error) {
	if strings.Index(target, "#") < 0 {
		return "", nil, &TagNotFoundError{Target: target}
	}

	entry, ok := d.queries[target]
	if !ok {
		return "", nil, &TagNotFoundError{Target: target}
	}

	if params != nil {
		if reflect.TypeOf(params).Kind() == reflect.Map {
			return "", nil, &InvalidParamsError{Message: "map is not allowed"}
		}
	}

	sql, args, err := builder.Execute(entry.Node.Body, params, entry.TypeInfo)
	if err != nil {
		return "", nil, err
	}
	return sql, args, nil
}
