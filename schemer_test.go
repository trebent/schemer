package schemer

import (
	"os"
	"testing"

	"github.com/xeipuuv/gojsonschema"
)

type (
	SmallCfg struct {
		Property1 string `json:"property1"`
		Property2 int    `json:"property2"`
	}
)

func TestParse(t *testing.T) {
	schema, err := os.ReadFile("testdata/schemas/small_schema.json")
	checkErr(err, t)

	loader := gojsonschema.NewBytesLoader(schema)
	schemer := New(loader)

	data, err := os.ReadFile("testdata/data/small_data.json")
	checkErr(err, t)
	schemer.Load(data)

	target := &SmallCfg{}
	checkErr(schemer.Parse(target), t)

	if target.Property1 != "Hello, World!" {
		t.Errorf("Expected Property1 to be 'Hello, World!', got '%s'", target.Property1)
	}
	if target.Property2 != 0 {
		t.Errorf("Expected Property2 to be 0, got %d", target.Property2)
	}
}
