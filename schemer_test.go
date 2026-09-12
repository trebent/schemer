package schemer

import (
	"testing"
)

func TestParse(t *testing.T) {
	loader := smallSchema(t)
	data := readData("testdata/data/small_data.json", t)

	schemer := New(loader)
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

func TestPathRef(t *testing.T) {
	t.Run("path ref cut short", func(t *testing.T) {
		loader := smallSchema(t)
		data := readData("testdata/data/pathref_cut_short.json", t)

		schemer := New(loader)
		schemer.Load(data)

		err := schemer.Parse(&SmallCfg{})
		if err == nil {
			t.Fatal("Expected an error, got nil")
		}
	})

	t.Run("path ref cut short 2", func(t *testing.T) {
		loader := smallSchema(t)
		data := readData("testdata/data/pathref_cut_short_2.json", t)

		schemer := New(loader)
		schemer.Load(data)

		err := schemer.Parse(&SmallCfg{})
		if err == nil {
			t.Fatal("Expected an error, got nil")
		}
	})

	t.Run("path ref string", func(t *testing.T) {
		loader := smallSchema(t)
		data := readData("testdata/data/pathref_string.json", t)

		schemer := New(loader)
		schemer.Load(data)

		target := &SmallCfg{}
		checkErr(schemer.Parse(target), t)

		if target.Property1 != target.Property3 {
			t.Fatalf("Property1 %s should match property3 %s", target.Property1, target.Property3)
		}
	})

	t.Run("path ref int", func(t *testing.T) {
		loader := smallSchema(t)
		data := readData("testdata/data/pathref_int.json", t)

		schemer := New(loader)
		schemer.Load(data)

		target := &SmallCfg{}
		checkErr(schemer.Parse(target), t)

		if target.Property2 != target.Property4 {
			t.Fatalf("Property2 %d should match property4 %d", target.Property2, target.Property4)
		}
	})

	t.Run("path ref object", func(t *testing.T) {
		loader := biggerSchema(t)
		data := readData("testdata/data/pathref_object.json", t)

		schemer := New(loader)
		schemer.Load(data)

		target := &ObjCfg{}
		err := schemer.Parse(target)
		if err == nil {
			t.Fatal("Object references are not supported, this should have failed")
		}
	})
}

func TestSupporting(t *testing.T) {
	t.Run("supporting schema 1, basic", func(t *testing.T) {
		loader := referenceSchema(t)
		supportingLoader := supportingSchema1(t)
		data := readData("testdata/data/supporting_1_basic.json", t)

		schemer := New(loader, supportingLoader)
		schemer.Load(data)

		target := &ReferencingCfg{}
		checkErr(schemer.Parse(target), t)
	})

	t.Run("supporting schema 1, extra", func(t *testing.T) {
		loader := referenceSchema(t)
		supportingLoader := supportingSchema1(t)
		data := readData("testdata/data/supporting_1_extra.json", t)

		schemer := New(loader, supportingLoader)
		schemer.Load(data)

		target := &ReferencingCfg{}
		err := schemer.Parse(target)
		if err == nil {
			t.Fatal("Extra properties should not be allowed")
		}
	})
}

func TestEnvRef(t *testing.T) {
	t.Run("env ref string", func(t *testing.T) {
		t.Setenv("EXISTS", "testing")

		loader := smallSchema(t)
		data := readData("testdata/data/envref.json", t)

		schemer := New(loader)
		schemer.Load(data)

		target := &SmallCfg{}
		checkErr(schemer.Parse(target), t)

		if target.Property3 != "testing" {
			t.Fatalf("Env variable did not get set into config, got %s", target.Property3)
		}
	})

	t.Run("env ref default int", func(t *testing.T) {
		loader := smallSchema(t)
		data := readData("testdata/data/envref_default.json", t)

		schemer := New(loader)
		schemer.Load(data)

		target := &SmallCfg{}
		checkErr(schemer.Parse(target), t)

		if target.Property2 != 666 {
			t.Fatalf("Env variable did not get default value set into config, got %d", target.Property2)
		}
	})
}
