package openapi

import (
	"context"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPIContractIsValid(t *testing.T) {
	loader := openapi3.NewLoader()

	document, err := loader.LoadFromFile(
		"../../cmd/api/openapi.yaml",
	)
	if err != nil {
		t.Fatalf("load OpenAPI contract: %v", err)
	}

	if err := document.Validate(context.Background()); err != nil {
		t.Fatalf("validate OpenAPI contract: %v", err)
	}
}
