package codegen

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rawBodyTestSpec has two POST operations with an identical JSON body. Only one of them is
// opted in to raw-body generation, so the other doubles as the control.
const rawBodyTestSpec = `
openapi: 3.0.0
info:
  title: Raw body test
  version: 1.0.0
paths:
  /webhooks/{provider}:
    post:
      operationId: handleProviderWebhook
      parameters:
        - name: provider
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
      responses:
        "202":
          description: accepted
  /things:
    post:
      operationId: createThing
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
      responses:
        "201":
          description: created
`

func generateWithRawBodyIDs(t *testing.T, rawBodyIDs []string) (string, error) {
	t.Helper()

	loader := openapi3.NewLoader()
	swagger, err := loader.LoadFromData([]byte(rawBodyTestSpec))
	require.NoError(t, err, "load spec")

	return Generate(swagger, Configuration{
		PackageName: "testrawbody",
		Generate: GenerateOptions{
			ChiServer: true,
			Models:    true,
		},
		OutputOptions: OutputOptions{
			RawBodyOperationIDs: rawBodyIDs,
		},
	})
}

func TestRawBodyOperationIDs(t *testing.T) {
	t.Run("omits the body type for an opted-in operation but keeps it for others", func(t *testing.T) {
		code, err := generateWithRawBodyIDs(t, []string{"handleProviderWebhook"})
		require.NoError(t, err)

		assert.NotContains(t, code, "HandleProviderWebhookJSONBody",
			"a raw-body operation must generate no request body type")
		assert.NotContains(t, code, "HandleProviderWebhookJSONRequestBody",
			"a raw-body operation must generate no request body alias")

		// The control operation is untouched: opting one operation in must not disable
		// body generation globally.
		assert.Contains(t, code, "CreateThingJSONRequestBody",
			"a non-opted-in operation must still generate its request body type")
	})

	t.Run("generates the body type when the operation is not opted in", func(t *testing.T) {
		code, err := generateWithRawBodyIDs(t, nil)
		require.NoError(t, err)

		assert.Contains(t, code, "HandleProviderWebhookJSONRequestBody",
			"without the opt-in the body type is generated as usual")
	})

	// Either spelling of the id opts the operation in, so a config written against the Go
	// name is not a silent no-op.
	t.Run("matches the normalized Go name as well as the spec operation-id", func(t *testing.T) {
		code, err := generateWithRawBodyIDs(t, []string{"HandleProviderWebhook"})
		require.NoError(t, err)

		assert.NotContains(t, code, "HandleProviderWebhookJSONRequestBody",
			"the normalized Go name must opt the operation in")
		assert.Contains(t, code, "CreateThingJSONRequestBody")
	})

	// Consistent with include-operation-ids / exclude-operation-ids: an id that matches
	// nothing is ignored rather than failing generation.
	t.Run("ignores operation-ids that match nothing", func(t *testing.T) {
		code, err := generateWithRawBodyIDs(t, []string{"nope", "alsoNope"})
		require.NoError(t, err, "an unknown operation-id must not fail generation")

		assert.Contains(t, code, "HandleProviderWebhookJSONRequestBody",
			"no operation is opted in, so every body type is generated as usual")
		assert.Contains(t, code, "CreateThingJSONRequestBody")
	})
}
