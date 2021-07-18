package jsonschema

import (
	"context"
	"cosmos"
	_ "embed"
	"fmt"
	"strings"

	"github.com/iancoleman/orderedmap"
	jsoniter "github.com/json-iterator/go"
	"github.com/mitchellh/mapstructure"
	js "github.com/xeipuuv/gojsonschema"
)

var (
	json                       = jsoniter.ConfigDefault
	_    cosmos.MessageService = (*MessageService)(nil)
)

//go:embed protocol.json
var protocol []byte

type MessageService struct {
	protocol *js.Schema
}

func NewMessageService() *MessageService {
	protocol, _ := js.NewSchema(js.NewBytesLoader(protocol))
	return &MessageService{protocol: protocol}
}

// Map is a utility type with additional methods useful for parsing json schema.
type Map struct {
	orderedmap.OrderedMap
}

func (m Map) M(key string) *Map {
	val, _ := m.Get(key)
	v, _ := val.(orderedmap.OrderedMap)
	return &Map{v}
}

func (m *Map) S(key string) string {
	val, _ := m.Get(key)
	v, _ := val.(string)
	return v
}

func (m *Map) AM(key string) []*Map {
	result := []*Map{}
	val, _ := m.Get(key)
	arr, _ := val.([]interface{})
	for _, x := range arr {
		if v, ok := x.(orderedmap.OrderedMap); !ok {
			return nil
		} else {
			result = append(result, &Map{v})
		}
	}
	return result
}

func (m *Map) AS(key string) []string {
	result := []string{}
	val, _ := m.Get(key)
	arr, _ := val.([]interface{})
	for _, x := range arr {
		if v, ok := x.(string); !ok {
			return nil
		} else {
			result = append(result, v)
		}
	}
	return result
}

func (m *Map) GoMap() map[string]interface{} {
	result := map[string]interface{}{}
	for _, key := range m.Keys() {
		if v, ok := m.Get(key); ok {
			result[key] = v
		}
	}
	return result
}

func (s *MessageService) CreateMessage(ctx context.Context, raw []byte) (*cosmos.Message, error) {
	rawLoader := js.NewBytesLoader(raw)

	result, err := s.protocol.Validate(rawLoader)
	if err != nil {
		return nil, err
	}

	if !result.Valid() {
		msg := strings.Builder{}
		for _, e := range result.Errors() {
			if !strings.HasPrefix(e.Description(), "Must validate") {
				msg.WriteString(fmt.Sprintf("%s\n\n", e))
			}
		}
		return nil, cosmos.Errorf(cosmos.EINVALID, msg.String())
	}

	msg := &cosmos.Message{}
	if err := json.Unmarshal(raw, msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal into cosmos.Message after JSON Schema was validated")
	}

	return msg, nil
}

func (s *MessageService) Validate(ctx context.Context, raw interface{}, message *cosmos.Message) error {
	var rawLoader js.JSONLoader
	var schemaLoader js.JSONLoader

	switch r := raw.(type) {
	case []byte:
		rawLoader = js.NewBytesLoader(r)
	case map[string]interface{}:
		rawLoader = js.NewGoLoader(r)
	default:
		panic("Unhandled type in Validate")
	}

	switch message.Type {
	case cosmos.MessageTypeSpec:
		schemaLoader = js.NewGoLoader(message.Spec.ConnectionSpecification)
	default:
		panic("Unhandled message type in JSON Schema validation switch")
	}

	result, err := js.Validate(schemaLoader, rawLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		msg := strings.Builder{}
		for _, e := range result.Errors() {
			if !strings.HasPrefix(e.Description(), "Must validate") {
				msg.WriteString(fmt.Sprintf("%s\n\n", e))
			}
		}
		return cosmos.Errorf(cosmos.EINVALID, msg.String())
	}

	return nil
}

func (s *MessageService) MessageToForm(ctx context.Context, message *cosmos.Message, additionalInfo interface{}) *cosmos.Form {
	switch message.Type {

	case cosmos.MessageTypeSpec:
		result := []*cosmos.FormFieldSpec{}
		parseConnectionSpec(&Map{message.Spec.ConnectionSpecification}, nil, &result, nil, "")

		// Set DependsOnIdx and DependsOnValue without modifying path.
		for _, r := range result {
			if r.OneOfKey && countOneOfPatterns(r.Path) > 1 {
				oneOfKeyIdx := getOneOfKeyIdx(r.Path, result, "parent")
				r.DependsOnIdx = &oneOfKeyIdx
				r.DependsOnValue = result[oneOfKeyIdx].Enum
			}
			if !r.OneOfKey && countOneOfPatterns(r.Path) > 0 {
				oneOfKeyIdx := getOneOfKeyIdx(r.Path, result, "sibling")
				r.DependsOnIdx = &oneOfKeyIdx
				r.DependsOnValue = result[oneOfKeyIdx].Enum
			}
		}

		// Remove the last occurrence of oneOfPattern from all paths.
		for _, r := range result {
			r.Path = getCompressedPath(r.Path)
		}

		// 1. Merge the enum's for oneOf keys which have the exact same path into a
		//    single entry.
		// 2. Mark all but one of the oneOf keys as "ignored" so that they are not
		//    displayed on the UI.
		// 3. If any entry depends on "ignored" entries, adjust their DependsOnIdx.
		for i, r := range result {
			if !r.OneOfKey || r.Ignore {
				continue
			}

			idx := findSimilarEntries(i, result)
			for _, j := range idx {
				r.Enum = append(r.Enum, result[j].Enum...)
				result[j].Ignore = true

				for _, r := range result {
					if r.DependsOnIdx != nil && *r.DependsOnIdx == j {
						i := i
						r.DependsOnIdx = &i
					}
				}
			}
		}

		return &cosmos.Form{Type: cosmos.FormTypeSpec, Spec: result}

	case cosmos.MessageTypeCatalog:
		result := []*cosmos.FormFieldCatalog{}
		supportedDestinationSyncModes, _ := additionalInfo.([]string)

		for _, stream := range message.Catalog.Streams {
			var field cosmos.FormFieldCatalog

			field.Stream = stream
			field.StreamName = stream.Name
			field.IsStreamSelected = true

			if stream.IsSyncModeAvailable(cosmos.SyncModeFullRefresh) {
				if contains(supportedDestinationSyncModes, cosmos.DestinationSyncModeOverwrite) {
					field.SyncModes = append(
						field.SyncModes,
						[]string{cosmos.SyncModeFullRefresh, cosmos.DestinationSyncModeOverwrite},
					)
				}
				if contains(supportedDestinationSyncModes, cosmos.DestinationSyncModeAppend) {
					field.SyncModes = append(
						field.SyncModes,
						[]string{cosmos.SyncModeFullRefresh, cosmos.DestinationSyncModeAppend},
					)
				}
			}

			if stream.IsSyncModeAvailable(cosmos.SyncModeIncremental) {
				if contains(supportedDestinationSyncModes, cosmos.DestinationSyncModeAppend) {
					field.SyncModes = append(
						field.SyncModes,
						[]string{cosmos.SyncModeIncremental, cosmos.DestinationSyncModeAppend},
					)
				}
				if contains(supportedDestinationSyncModes, cosmos.DestinationSyncModeAppendDedup) {
					field.SyncModes = append(
						field.SyncModes,
						[]string{cosmos.SyncModeIncremental, cosmos.DestinationSyncModeAppendDedup},
					)
				}
				if contains(supportedDestinationSyncModes, cosmos.DestinationSyncModeUpsertDedup) {
					field.SyncModes = append(
						field.SyncModes,
						[]string{cosmos.SyncModeIncremental, cosmos.DestinationSyncModeUpsertDedup},
					)
				}

			}

			field.SelectedSyncMode = field.SyncModes[0]

			if stream.IsSyncModeAvailable(cosmos.SyncModeIncremental) {
				if stream.SourceDefinedCursor {
					if len(stream.DefaultCursorField) != 0 {
						field.CursorFields = [][]string{stream.DefaultCursorField}
					} else {
						field.CursorFields = [][]string{{"source-defined cursor"}}
					}
					field.SelectedCursorField = field.CursorFields[0]
				} else {
					parseStream(&Map{stream.JSONSchema}, nil, &field.CursorFields)
					if len(stream.DefaultCursorField) != 0 {
						field.SelectedCursorField = stream.DefaultCursorField
					}
				}
			}

			if contains(supportedDestinationSyncModes, cosmos.DestinationSyncModeAppendDedup) ||
				contains(supportedDestinationSyncModes, cosmos.DestinationSyncModeUpsertDedup) {
				if len(stream.SourceDefinedPrimaryKey) != 0 {
					field.PrimaryKeys = stream.SourceDefinedPrimaryKey
					field.SelectedPrimaryKey = stream.SourceDefinedPrimaryKey
				} else {
					parseStream(&Map{stream.JSONSchema}, nil, &field.PrimaryKeys)
				}
			}

			result = append(result, &field)
		}

		return &cosmos.Form{Type: cosmos.FormTypeCatalog, Catalog: result}

	default:
		panic("Unhandled message type in MessageToForm")
	}
}

func parseConnectionSpec(schema *Map, path []string, result *[]*cosmos.FormFieldSpec, oneOfKey []string, oneOfTitle string) {
	properties := schema.M(js.KEY_PROPERTIES)
	required := schema.AS(js.KEY_REQUIRED)
	oneOf := schema.AM(js.KEY_ONE_OF)

	for _, key := range properties.Keys() {
		value := properties.M(key)

		switch v, _ := value.Get(js.KEY_TYPE); getJsType(v) {
		case js.TYPE_INTEGER, js.TYPE_NUMBER, js.TYPE_STRING, js.TYPE_BOOLEAN, js.TYPE_ARRAY:
			field := &cosmos.FormFieldSpec{}

			mapstructure.WeakDecode(value.GoMap(), field)
			field.Path = append(path, key)
			field.Value = field.Default

			if field.Type == js.TYPE_ARRAY {
				items := value.M(js.KEY_ITEMS)
				mapstructure.WeakDecode(items.GoMap(), field)
				if field.Type == js.TYPE_OBJECT {
					panic("Array of objects is not supported in connection specification")
				}
				if len(field.Enum) == 0 && field.Const == nil {
					panic("Non-enum array is not supported in connection specification")
				}
				field.Multiple = true
			}

			for _, req_field := range required {
				if key == req_field {
					field.Required = true
				}
			}

			// Transfer contents of const into enum.
			if field.Const != nil {
				field.Enum = []interface{}{field.Const}
				field.Const = nil
			}

			// Mark the oneOfKey as such.
			if testEq(field.Path, oneOfKey) {
				field.OneOfKey = true
				field.Title = oneOfTitle
			}

			*result = append(*result, field)

		case js.TYPE_OBJECT:
			parseConnectionSpec(value, append(path, key), result, oneOfKey, oneOfTitle)

		case js.TYPE_NULL:

		default:
			panic("Unhandled JSON Schema type in parseConnectionSpec")
		}
	}

	for i, one := range oneOf {
		i := fmt.Sprintf("<<%d>>", i)
		oneOfKey := append(path, i, getOneOfKey(oneOf))
		oneOfTitle := schema.S(js.KEY_TITLE)
		parseConnectionSpec(one, append(path, i), result, oneOfKey, oneOfTitle)
	}
}

func parseStream(schema *Map, path []string, paths *[][]string) {
	properties := schema.M(js.KEY_PROPERTIES)

	for _, key := range properties.Keys() {
		value := properties.M(key)

		switch v, _ := value.Get(js.KEY_TYPE); getJsType(v) {
		case js.TYPE_INTEGER, js.TYPE_NUMBER, js.TYPE_STRING, js.TYPE_BOOLEAN:
			*paths = append(*paths, append(path, key))

		case js.TYPE_ARRAY:
			// array fields cannot be used as cursor or primary key.

		case js.TYPE_OBJECT:
			parseStream(value, append(path, key), paths)

		case js.TYPE_NULL:

		default:
			panic("Unhandled JSON Schema type in parseStream")
		}
	}
}

func contains(arr []string, element string) bool {
	for _, item := range arr {
		if item == element {
			return true
		}
	}
	return false
}

func getJsType(item interface{}) string {
	// If the type is a string, return the same string.
	// If the type is an array, return the first non-null type.
	switch jstype := item.(type) {
	case string:
		return jstype
	case []interface{}:
		for _, t := range jstype {
			if t.(string) == js.TYPE_NULL {
				continue
			}
			return t.(string)
		}
	}
	return js.TYPE_NULL
}

// getOneOfKey returns the key for the oneOf which is essentially the "required" key
// with the highest frequency among all the oneOf's. The assumption is that there is
// only one such key.
func getOneOfKey(oneOf []*Map) string {
	freqMap := map[string]int{}
	for _, one := range oneOf {
		requiredKeys := one.AS(js.KEY_REQUIRED)
		for _, k := range requiredKeys {
			freqMap[k] = freqMap[k] + 1
		}
	}

	maxFreq := 0
	mostFrequentKey := ""
	for k, v := range freqMap {
		if v > maxFreq {
			maxFreq = v
			mostFrequentKey = k
		}
	}

	return mostFrequentKey
}

func testEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func countOneOfPatterns(path []string) int {
	count := 0
	for _, p := range path {
		if cosmos.OneOfPattern.MatchString(p) {
			count++
		}
	}
	return count
}

// getOneOfKeyIdx returns the index of the oneOf key for the given path. If the
// given path itself is a oneOf key, return its parent oneOf key (if any).
func getOneOfKeyIdx(path []string, result []*cosmos.FormFieldSpec, relationship string) int {
	idx := []int{}
	for i, p := range path {
		if cosmos.OneOfPattern.MatchString(p) {
			idx = append(idx, i)
		}
	}

	var prefix []string
	switch relationship {
	case "parent":
		prefix = path[:idx[len(idx)-2]+1]
	case "sibling":
		prefix = path[:idx[len(idx)-1]+1]
	default:
		panic("Unknown relationship in getOneOfKeyIdx")
	}

	for i, r := range result {
		if !r.OneOfKey {
			continue
		}
		if !strings.HasPrefix(strings.Join(r.Path, "/"), strings.Join(prefix, "/")) {
			continue
		}
		if len(r.Path)-len(prefix) != 1 {
			continue
		}
		return i
	}

	panic(fmt.Sprintf("We should always find the %s oneOf key index", relationship))
}

// getCompressedPath returns the path after removing just the last occurrence
// of the oneOfPattern (if any).
func getCompressedPath(path []string) []string {
	idx := []int{}
	for i, p := range path {
		if cosmos.OneOfPattern.MatchString(p) {
			idx = append(idx, i)
		}
	}

	if len(idx) == 0 {
		return path
	} else {
		lastIdx := idx[len(idx)-1]
		return append(path[:lastIdx], path[lastIdx+1:]...)
	}
}

// findSimilarEntries returns the indices of entries (excluding the current
// one) with exactly the same path.
func findSimilarEntries(curIdx int, result []*cosmos.FormFieldSpec) []int {
	idx := []int{}
	for i, r := range result {
		if i == curIdx {
			continue
		}
		if testEq(result[curIdx].Path, r.Path) {
			idx = append(idx, i)
		}
	}
	return idx
}
