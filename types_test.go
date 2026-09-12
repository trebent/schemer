package schemer

type (
	SmallCfg struct {
		Property1 string `json:"property1"`
		Property2 int    `json:"property2"`
		Property3 string `json:"property3"`
		Property4 int    `json:"property4"`
	}

	ObjCfg struct {
		Obj1 basicObject `json:"obj1"`
		Obj2 basicObject `json:"obj2"`
	}
	basicObject struct {
		String string `json:"string"`
		Int    int    `json:"int"`
	}

	ReferencingCfg struct {
		Property1   string         `json:"property1"`
		Referencing SupportingCfg1 `json:"referencing"`
	}
	SupportingCfg1 struct {
		Supporting1 string `json:"supporting1"`
		Supporting2 int    `json:"supporting2"`
	}
)
