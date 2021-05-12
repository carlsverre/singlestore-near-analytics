package src_test

import (
	"testing"

	. "f0a.org/singlestore-near-analytics/src"
)

func TestAvroSchema(t *testing.T) {
	for _, model := range Models {
		schema, err := GenerateAvroSchema(model)
		if err != nil {
			t.Errorf("failed to generate avro schema for %+v: %+v", model, err)
		}
		t.Logf("schema: %s", schema.String())
	}
}
