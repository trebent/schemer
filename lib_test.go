package schemer

import (
	"os"
	"testing"

	"github.com/xeipuuv/gojsonschema"
)

func checkErr(err error, t *testing.T) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func readData(path string, t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	checkErr(err, t)
	return data
}

func smallSchema(t *testing.T) gojsonschema.JSONLoader {
	t.Helper()
	schema, err := os.ReadFile("testdata/schemas/small_schema.json")
	checkErr(err, t)

	loader := gojsonschema.NewBytesLoader(schema)
	return loader
}

func biggerSchema(t *testing.T) gojsonschema.JSONLoader {
	t.Helper()
	schema, err := os.ReadFile("testdata/schemas/bigger_schema.json")
	checkErr(err, t)

	loader := gojsonschema.NewBytesLoader(schema)
	return loader
}

func referenceSchema(t *testing.T) gojsonschema.JSONLoader {
	t.Helper()
	schema, err := os.ReadFile("testdata/schemas/reference_schema.json")
	checkErr(err, t)

	loader := gojsonschema.NewBytesLoader(schema)
	return loader
}

func supportingSchema1(t *testing.T) gojsonschema.JSONLoader {
	t.Helper()
	schema, err := os.ReadFile("testdata/schemas/supporting_schema_1.json")
	checkErr(err, t)

	loader := gojsonschema.NewBytesLoader(schema)
	return loader
}
