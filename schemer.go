package schemer

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/trebent/zerologr"
	"github.com/xeipuuv/gojsonschema"
)

type (
	// Schemer takes as input a main schema and supporting schemas, and can load JSON data to validate against the schemas.
	Schemer struct {
		// MainSchema is the main schema to validate against.
		MainSchema gojsonschema.JSONLoader
		// SupportingSchemas are the supporting schemas to validate against.
		SupportingSchemas []gojsonschema.JSONLoader

		data        []byte
		escapedData []byte

		values map[string]any
		refs   map[string]string
	}
)

var (
	envRe  = regexp.MustCompile(`\$\{env:([a-zA-Z0-9_:]+)\}`)
	pathRe = regexp.MustCompile(`\$\{ref:([a-zA-Z0-9_\.\[\]:]+)\}`)
)

// New creates a new Schemer instance with the provided supporting schemas and main schema.
func New(
	mainSchema gojsonschema.JSONLoader,
	supportingSchemas ...gojsonschema.JSONLoader,
) *Schemer {
	return &Schemer{
		MainSchema:        mainSchema,
		SupportingSchemas: supportingSchemas,
		values:            make(map[string]any),
		refs:              make(map[string]string),
	}
}

func (sch *Schemer) Load(data []byte) {
	sch.data = data
}

func (sch *Schemer) Parse(target any) error {
	if err := sch.resolveReferences(); err != nil {
		return err
	}

	if err := sch.validateSchema(); err != nil {
		return err
	}

	if err := sch.loadData(target); err != nil {
		return err
	}

	// Free allocated memory for intermediate data structures.
	sch.data = nil
	sch.escapedData = nil
	sch.values = nil
	sch.refs = nil

	return nil
}

func (sch *Schemer) resolveReferences() error {
	if err := sch.escapeReferences(); err != nil {
		return err
	}

	if err := sch.walkForReferences(); err != nil {
		return err
	}

	if err := sch.findReferenceValues(); err != nil {
		return err
	}

	if err := sch.replaceReferencesInData(); err != nil {
		return err
	}

	return nil
}

func (sch *Schemer) escapeReferences() error {
	zerologr.V(100).Info("Escaping references")
	sch.escapedData = sch.data
	i := 0
	for i < len(sch.data) {
		//nolint:gocritic // ignore: nestedIfs
		if sch.data[i] == '$' && isReference(sch.data[i:i+6]) && sch.data[i-1] != '"' {
			zerologr.V(100).Info(
				fmt.Sprintf("Found unescaped reference at index %d", i),
			)

			end := bytes.IndexByte(sch.data[i:], '}')
			if end == -1 {
				return errors.New("malformed reference: missing closing '}'")
			}

			escapedRef := append([]byte{'"'}, sch.data[i:i+end+1]...)
			escapedRef = append(escapedRef, '"')

			zerologr.V(100).Info("Escaped reference: " + string(escapedRef))

			sch.escapedData = bytes.Replace(sch.escapedData, sch.data[i:i+end+1], escapedRef, 1)

			zerologr.V(100).Info("Intermediate escaped data: \n" + string(sch.escapedData))

			i = i + end + 3
		} else if sch.data[i] == '$' && isReference(sch.data[i:i+6]) {
			zerologr.V(100).Info(
				fmt.Sprintf("Found already escaped reference at index %d", i),
			)

			end := bytes.IndexByte(sch.data[i:], '}')
			if end == -1 {
				return errors.New("malformed reference: missing closing '}'")
			}

			i = i + end + 2
		} else {
			i++
		}
	}

	zerologr.V(100).Info("Escaped references")

	return nil
}

func (sch *Schemer) walkForReferences() error {
	if len(sch.data) == 0 {
		return nil
	}

	zerologr.V(100).Info("Gathering references")

	generic := make(map[string]any)
	if err := json.Unmarshal(sch.escapedData, &generic); err != nil {
		return err
	}

	if err := sch.walk("", generic); err != nil {
		return err
	}

	zerologr.V(100).Info("Gathered references")
	zerologr.V(100).Info(fmt.Sprintf("Current values map: %+v", sch.values))
	zerologr.V(100).Info(fmt.Sprintf("Current refs map: %+v", sch.refs))

	return nil
}

func (sch *Schemer) walk(currentPath string, generic any) error {
	/*
		Walk through a JSON object and find final values for all JSON paths.

		For each found value, store the path and the value it contains.
		For each found reference, store the reference path.
		Enabled to replace the reference path with the actual value.
	*/
	zerologr.V(100).Info("Walking for references in path '" + currentPath + "'")

	switch val := generic.(type) {
	case map[string]any:
		zerologr.V(100).Info("Walking into map in path '" + currentPath + "'")

		for k, v := range val {
			zerologr.V(100).Info("Key: " + k + ", Value: " + fmt.Sprint(v))

			newPath := currentPath
			if newPath != "" {
				newPath += "."
			}
			newPath += k

			if err := sch.walk(newPath, v); err != nil {
				return err
			}
		}
	case []any:
		zerologr.V(100).Info("Walking into array in path '" + currentPath + "'")

		for i, item := range val {
			if err := sch.walk(currentPath+"["+strconv.Itoa(i)+"]", item); err != nil {
				return err
			}
		}
	case string:
		if isReference([]byte(val)) {
			zerologr.V(100).Info("Found ref: " + val)
			sch.refs[val] = ""
		}
		sch.values[currentPath] = val
	default:
		zerologr.V(100).Info("Storing final value for path '" + currentPath + "'")
		sch.values[currentPath] = val
	}

	return nil
}

func (sch *Schemer) findReferenceValues() error {
	zerologr.V(100).Info("Finding reference values")
	/*
		Values contain values for full paths, but some contains references as well. Now what's needed is:

		For values for a given path: return the value if it's not a reference.
		For references: find if the reference can be walked to a value. Environment variables can be resolved
		immediately, path references need to be walked through the values map until a final value is found.

		Environment variable references are resolved first, then path references.
	*/
	for ref := range sch.refs {
		var err error
		if isEnvReference(ref) {
			zerologr.V(100).Info("Resolving environment reference value for: " + ref)
			sch.refs[ref], err = getEnvReferenceValue(ref)
			if err != nil {
				return err
			}
		}
	}

	zerologr.V(100).Info(fmt.Sprintf("Realised env refs: %s", sch.refs))

	for ref := range sch.refs {
		var err error
		if isPathReference(ref) {
			zerologr.V(100).Info("Resolving path reference value for: " + ref)

			// Find if the path reference can be walked to a final value
			sch.refs[ref], err = sch.findReferenceValue(ref)
			if err != nil {
				return err
			}

			zerologr.V(100).
				Info("Resolved path reference value for: " + ref + ", value: " + sch.refs[ref])
		}
	}

	zerologr.V(100).Info(fmt.Sprintf("Realised all refs: %s", sch.refs))

	return nil
}

func (sch *Schemer) findReferenceValue(ref string) (string, error) {
	zerologr.V(100).Info("Finding value for path reference: " + ref)

	valuePath, err := getPathFromReference(ref)
	if err != nil {
		return "", err
	}

	value, ok := sch.values[valuePath]
	if !ok {
		return "", errors.New("referenced path was not found: " + valuePath)
	}

	decoded, ok := value.(string)
	if ok {
		if isEnvReference(decoded) {
			zerologr.V(100).Info("Nested environment reference found: " + decoded)
			return sch.refs[decoded], nil
		} else if isPathReference(decoded) {
			zerologr.V(100).Info("Nested path reference found, walking path: " + decoded)
			return sch.walkRefs(valuePath, decoded)
		}
	}

	return fmt.Sprint(value), nil
}

func (sch *Schemer) walkRefs(originPath, ref string) (string, error) {
	zerologr.V(100).Info("Walking references, origin: " + originPath + ", ref: " + ref)

	valuePath, err := getPathFromReference(ref)
	if err != nil {
		return "", err
	}

	if originPath == valuePath {
		return "", errors.New("cischular reference detected: " + originPath)
	}

	value, ok := sch.values[valuePath]
	if !ok {
		return "", errors.New("reference path not found: " + valuePath)
	}

	decoded, ok := value.(string)
	if ok {
		if isEnvReference(decoded) {
			zerologr.V(100).Info("Env ref found during walk: " + decoded)
			return sch.refs[decoded], nil
		} else if isPathReference(decoded) {
			zerologr.V(100).Info("Path ref found during walk: " + decoded)
			// Don't decode the reference path here, it's done recursively in the next call to walkRefs.
			// The next call to walkRefs will extract the path from the reference and compare it to the originPath.
			return sch.walkRefs(originPath, decoded)
		}
	}
	zerologr.V(100).
		Info("Final value found for " + originPath + " during walk: " + fmt.Sprint(value))

	return fmt.Sprint(value), nil
}

func (sch *Schemer) replaceReferencesInData() error {
	/*
		Replace all references in the original JSON data with their resolved values.
	*/
	if len(sch.data) == 0 {
		return nil
	}

	dataStr := string(sch.data)
	zerologr.V(100).Info("Replacing references: " + dataStr)

	for ref, val := range sch.refs {
		zerologr.V(100).Info(
			fmt.Sprintf("Replacing reference '%s' with value '%s'", ref, val),
		)
		dataStr = strings.ReplaceAll(dataStr, ref, val)
		zerologr.V(100).Info("Intermediate replaced data: " + dataStr)
	}

	sch.data = []byte(dataStr)

	zerologr.V(100).Info("Replaced all references: " + string(sch.data))

	return nil
}

func (sch *Schemer) validateSchema() error {
	zerologr.V(100).Info("Validating schema")

	if len(sch.data) == 0 {
		return nil
	}

	sl := gojsonschema.NewSchemaLoader()
	sl.AutoDetect = false
	sl.Validate = true
	sl.Draft = gojsonschema.Draft7
	if err := sl.AddSchemas(sch.SupportingSchemas...); err != nil {
		zerologr.Error(err, "Failed to add global schemas")
		return err
	}

	compiledSchema, err := sl.Compile(sch.MainSchema)
	if err != nil {
		zerologr.Error(err, "Failed to compile root schema")
		return err
	}

	result, err := compiledSchema.Validate(gojsonschema.NewBytesLoader(sch.data))
	if err != nil {
		return err
	}

	if !result.Valid() {
		var fullError error
		for _, validationErr := range result.Errors() {
			fullError = fmt.Errorf(
				"%w, %s - %s",
				fullError,
				validationErr.Field(),
				validationErr.Description(),
			)
		}

		return fmt.Errorf("schema validation failed: %w", fullError)
	}

	return nil
}

func (sch *Schemer) loadData(target any) error {
	return json.Unmarshal(sch.data, target)
}

func isReference(data []byte) bool {
	zerologr.V(100).Info("isReference check on: " + string(data))

	prefixes := [][]byte{[]byte("${env:"), []byte("${ref:")}

	for _, prefix := range prefixes {
		if bytes.HasPrefix(data, prefix) {
			return true
		}
	}

	return false
}

func isEnvReference(ref string) bool {
	return strings.HasPrefix(ref, "${env:")
}

func isPathReference(ref string) bool {
	return strings.HasPrefix(ref, "${ref:")
}

func getEnvReferenceValue(ref string) (string, error) {
	groups := envRe.FindStringSubmatch(ref)
	zerologr.V(100).Info("Found env ref submatch groups: ", "ref", ref, "groups", groups)
	if len(groups) < 2 {
		return "", errors.New("malformed env var reference: " + ref)
	}

	split := strings.Split(groups[1], ":")

	val, ok := os.LookupEnv(split[0])
	if !ok {
		if len(split) > 1 {
			zerologr.V(100).Info("Using default env var value: " + split[1])
			return split[1], nil
		}
		return "", errors.New("environment variable not found: " + split[0])
	}
	zerologr.V(100).Info("Found env var value: " + val)

	return val, nil
}

func getPathFromReference(ref string) (string, error) {
	groups := pathRe.FindStringSubmatch(ref)
	if len(groups) < 2 {
		return "", errors.New("malformed path reference: " + ref)
	}
	zerologr.V(100).Info("Extracted path from reference", "ref", ref, "path", groups[1])

	return groups[1], nil
}
