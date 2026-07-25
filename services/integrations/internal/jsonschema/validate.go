package jsonschema

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// ValidateDocument validates data against a JSON Schema document.
func ValidateDocument(schemaDoc, data json.RawMessage) error {
	if len(schemaDoc) == 0 {
		return fmt.Errorf("schema is required")
	}
	if len(data) == 0 {
		data = json.RawMessage(`{}`)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", strings.NewReader(string(schemaDoc))); err != nil {
		return fmt.Errorf("load schema: %w", err)
	}

	schema, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("compile schema: %w", err)
	}

	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("config must be valid JSON")
	}

	if err := schema.Validate(value); err != nil {
		return err
	}
	return nil
}
