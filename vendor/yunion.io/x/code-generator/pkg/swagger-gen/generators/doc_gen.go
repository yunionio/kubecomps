package generators

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"k8s.io/gengo/generator"
)

const swaggerMeta = `
// {{.Service}} API
//
//     Schemes: https, http
//     BasePath: /
//     Version: 1.0
//     Host: "127.0.0.1:8889"
//     Contact: Zexi Li<lizexi@yunion.cn>
//     License: Apache 2.0 http://www.apache.org/licenses/LICENSE-2.0.html
//
//     Consumes:
//     - application/json
//
//     Produces:
//     - application/json
//
//     SecurityDefinitions:
//     keystone:
//       name: X-Auth-Token
//       type: apiKey
//       in: header
//
// swagger:meta
`

type swaggerDocGen struct {
	generator.DefaultGen
}

func NewSwaggerDocGen() generator.Generator {
	return &swaggerDocGen{
		DefaultGen: generator.DefaultGen{
			OptionalName: "doc",
		},
	}
}

type DocPackage struct {
	*generator.DefaultPackage
}

func NewDocPackage(pkgName string, pkgPath string, header []byte, service string) generator.Package {
	out := new(bytes.Buffer)
	t := template.Must(template.New("compiled_template").Parse(swaggerMeta))
	if err := t.Execute(out, map[string]string{"Service": strings.Title(service)}); err != nil {
		panic(err)
	}
	defaultPkg := &generator.DefaultPackage{
		PackageName: pkgName,
		PackagePath: pkgPath,
		HeaderText:  []byte(fmt.Sprintf("%s %s", header, out.String())),
		GeneratorFunc: func(c *generator.Context) []generator.Generator {
			return []generator.Generator{
				// Always generate a "doc.go" file.
				NewSwaggerDocGen(),
			}
		},
	}
	return defaultPkg
}
