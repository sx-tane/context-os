package filesystem

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	metadataOpenAPITitle           = "openapi_title"
	metadataOpenAPIVersion         = "openapi_version"
	metadataOpenAPISpecVersion     = "openapi_spec_version"
	metadataOpenAPIContentHash     = "openapi_content_hash"
	metadataOpenAPIPathCount       = "openapi_path_count"
	metadataOpenAPIOperationCount  = "openapi_operation_count"
	metadataOpenAPISchemaCount     = "openapi_schema_count"
	metadataOpenAPIEnumCount       = "openapi_enum_count"
	metadataOpenAPIEndpointPaths   = "openapi_endpoint_pointers"
	metadataOpenAPISchemaPointers  = "openapi_schema_pointers"
	metadataOpenAPIEnumPointers    = "openapi_enum_pointers"
	metadataOpenAPISourceLocations = "openapi_source_locations"
)

type specExtraction struct {
	Title            string
	Version          string
	SpecVersion      string
	Hash             string
	PathCount        int
	OperationCount   int
	SchemaCount      int
	EnumCount        int
	EndpointPointers []string
	SchemaPointers   []string
	EnumPointers     []string
}

func extractOpenAPISpec(data []byte) (specExtraction, bool) {
	extracted := summarizeOpenAPISpec(data)
	return extracted, extracted.SpecVersion != ""
}

func summarizeOpenAPISpec(data []byte) specExtraction {
	summary := specExtraction{Hash: hashBytes(data)}
	doc, ok := parseSpecDocument(data)
	if !ok {
		return summary
	}

	if value, ok := stringAt(doc, "openapi"); ok {
		summary.SpecVersion = value
	} else if value, ok := stringAt(doc, "swagger"); ok {
		summary.SpecVersion = value
	}
	if info, ok := mapAt(doc, "info"); ok {
		summary.Title, _ = stringAt(info, "title")
		summary.Version, _ = stringAt(info, "version")
	}

	if paths, ok := mapAt(doc, "paths"); ok {
		summary.PathCount = len(paths)
		summary.EndpointPointers = collectEndpointPointers(paths)
		summary.OperationCount = len(summary.EndpointPointers)
	}

	summary.SchemaPointers = collectSchemaPointers(doc)
	summary.SchemaCount = len(summary.SchemaPointers)
	summary.EnumPointers = collectEnumPointers(doc)
	summary.EnumCount = len(summary.EnumPointers)
	return summary
}

func parseSpecDocument(data []byte) (map[string]any, bool) {
	var jsonDoc map[string]any
	if err := json.Unmarshal(data, &jsonDoc); err == nil {
		return jsonDoc, true
	}

	var yamlDoc any
	if err := yaml.Unmarshal(data, &yamlDoc); err != nil {
		return nil, false
	}
	converted, ok := normalizeYAML(yamlDoc).(map[string]any)
	return converted, ok
}

func normalizeYAML(value any) any {
	switch typed := value.(type) {
	case map[interface{}]interface{}:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			out[fmt.Sprint(key)] = normalizeYAML(child)
		}
		return out
	case []interface{}:
		out := make([]any, len(typed))
		for index, child := range typed {
			out[index] = normalizeYAML(child)
		}
		return out
	default:
		return value
	}
}

func collectEndpointPointers(paths map[string]any) []string {
	var pointers []string
	for _, pathName := range sortedKeys(paths) {
		pathItem, ok := paths[pathName].(map[string]any)
		if !ok {
			continue
		}
		for _, method := range sortedKeys(pathItem) {
			method = strings.ToLower(method)
			if !isOpenAPIMethod(method) {
				continue
			}
			pointers = append(pointers, "/paths/"+jsonPointerEscape(pathName)+"/"+method)
		}
	}
	return pointers
}

func isOpenAPIMethod(method string) bool {
	switch method {
	case "delete", "get", "head", "options", "patch", "post", "put", "trace":
		return true
	default:
		return false
	}
}

func collectSchemaPointers(doc map[string]any) []string {
	var schemas map[string]any
	if components, ok := mapAt(doc, "components"); ok {
		schemas, _ = mapAt(components, "schemas")
	}
	if schemas == nil {
		schemas, _ = mapAt(doc, "definitions")
	}
	if schemas == nil {
		return nil
	}

	base := "/components/schemas"
	if _, ok := doc["definitions"]; ok {
		base = "/definitions"
	}

	names := sortedKeys(schemas)
	pointers := make([]string, 0, len(names))
	for _, name := range names {
		pointers = append(pointers, base+"/"+jsonPointerEscape(name))
	}
	return pointers
}

func collectEnumPointers(doc map[string]any) []string {
	var pointers []string
	walkSpec(doc, "", func(pointer string, value any) {
		if pointer == "" || !strings.HasSuffix(pointer, "/enum") {
			return
		}
		if _, ok := value.([]any); ok {
			pointers = append(pointers, pointer)
		}
	})
	sort.Strings(pointers)
	return pointers
}

func walkSpec(value any, pointer string, visit func(string, any)) {
	visit(pointer, value)
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range sortedKeys(typed) {
			walkSpec(typed[key], pointer+"/"+jsonPointerEscape(key), visit)
		}
	case []any:
		for index, child := range typed {
			walkSpec(child, pointer+"/"+strconv.Itoa(index), visit)
		}
	}
}

func enrichOpenAPIMetadata(extracted specExtraction, metadata map[string]string) {
	setIfMissing(metadata, metadataOpenAPITitle, extracted.Title)
	setIfMissing(metadata, metadataOpenAPIVersion, extracted.Version)
	setIfMissing(metadata, metadataOpenAPISpecVersion, extracted.SpecVersion)
	setIfMissing(metadata, metadataOpenAPIContentHash, extracted.Hash)
	setIfMissing(metadata, metadataOpenAPIPathCount, strconv.Itoa(extracted.PathCount))
	setIfMissing(metadata, metadataOpenAPIOperationCount, strconv.Itoa(extracted.OperationCount))
	setIfMissing(metadata, metadataOpenAPISchemaCount, strconv.Itoa(extracted.SchemaCount))
	setIfMissing(metadata, metadataOpenAPIEnumCount, strconv.Itoa(extracted.EnumCount))
	setJSONMetadata(metadata, metadataOpenAPIEndpointPaths, extracted.EndpointPointers)
	setJSONMetadata(metadata, metadataOpenAPISchemaPointers, extracted.SchemaPointers)
	setJSONMetadata(metadata, metadataOpenAPIEnumPointers, extracted.EnumPointers)
	locations := append(append([]string{}, extracted.EndpointPointers...), extracted.SchemaPointers...)
	locations = append(locations, extracted.EnumPointers...)
	setJSONMetadata(metadata, metadataOpenAPISourceLocations, locations)
}

func setJSONMetadata(metadata map[string]string, key string, values []string) {
	if len(values) == 0 {
		return
	}
	if _, exists := metadata[key]; exists {
		return
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return
	}
	metadata[key] = string(encoded)
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func stringAt(values map[string]any, key string) (string, bool) {
	value, ok := values[key]
	if !ok {
		return "", false
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed), strings.TrimSpace(typed) != ""
	case fmt.Stringer:
		text := strings.TrimSpace(typed.String())
		return text, text != ""
	default:
		return "", false
	}
}

func mapAt(values map[string]any, key string) (map[string]any, bool) {
	value, ok := values[key]
	if !ok {
		return nil, false
	}
	typed, ok := value.(map[string]any)
	return typed, ok
}

func jsonPointerEscape(value string) string {
	value = strings.ReplaceAll(value, "~", "~0")
	return strings.ReplaceAll(value, "/", "~1")
}
