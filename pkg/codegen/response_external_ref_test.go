package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

// Response body $ref'd from another file whose own refs are relative to that
// file: the root config cannot map them, the type is imported, so generation
// must still succeed.
func TestStrictResponseExternalRefWithNestedRelativeRefs(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"admin/openapi.yaml": `
openapi: 3.0.0
info: {title: t, version: "1"}
paths:
  /accounts:
    get:
      operationId: getAccount
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                $ref: "./schemas/billing.yaml#/components/schemas/Account"
`,
		"admin/schemas/billing.yaml": `
openapi: 3.0.0
info: {title: t, version: "1"}
paths: {}
components:
  schemas:
    Account:
      type: object
      properties:
        accountId:
          $ref: "../../common/schemas/common.yaml#/components/schemas/UUID"
`,
		"common/schemas/common.yaml": `
openapi: 3.0.0
info: {title: t, version: "1"}
paths: {}
components:
  schemas:
    UUID: {type: string, format: uuid}
`,
	}
	for name, body := range files {
		p := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(body), 0o644))
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	swagger, err := loader.LoadFromFile(filepath.Join(dir, "admin/openapi.yaml"))
	require.NoError(t, err)

	for name, gen := range map[string]GenerateOptions{
		"strict-server":        {StdHTTPServer: true, Strict: true, Models: true},
		"strict-hybrid-server": {FiberServer: true, StrictHybrid: true, Models: true},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Generate(swagger, Configuration{
				PackageName: "router",
				Generate:    gen,
				ImportMapping: map[string]string{
					"./schemas/billing.yaml":        "example.com/dto",
					"../common/schemas/common.yaml": "example.com/shared",
				},
			})
			require.NoError(t, err)
		})
	}
}
