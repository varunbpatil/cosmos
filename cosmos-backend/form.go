package cosmos

import (
	"regexp"
)

const (
	FormTypeSpec    = "SPEC"
	FormTypeCatalog = "CATALOG"
)

var (
	OneOfPattern = regexp.MustCompile(`^<<\d+>>$`)
)

type Form struct {
	Type    string              `json:"type,omitempty"`
	Spec    []*FormFieldSpec    `json:"spec,omitempty"`
	Catalog []*FormFieldCatalog `json:"catalog,omitempty"`
}

type FormFieldSpec struct {
	Path           []string      `json:"path"`
	Title          string        `json:"title"       mapstructure:"title"`
	Description    string        `json:"description" mapstructure:"description"`
	Default        interface{}   `json:"default"     mapstructure:"default"`
	Examples       []interface{} `json:"examples"    mapstructure:"examples"`
	Type           string        `json:"type"        mapstructure:"type"`
	Enum           []interface{} `json:"enum"        mapstructure:"enum"`
	Const          interface{}   `json:"const"       mapstructure:"const"`
	Secret         bool          `json:"secret"      mapstructure:"airbyte_secret"`
	Order          int           `json:"order"       mapstructure:"order"`
	Value          interface{}   `json:"value"`
	Multiple       bool          `json:"multiple"`
	Required       bool          `json:"required"`
	DependsOnIdx   *int          `json:"dependsOnIdx"`
	DependsOnValue []interface{} `json:"dependsOnValue"`
	OneOfKey       bool          `json:"oneOfKey"`
	Ignore         bool          `json:"ignore"`
}

type FormFieldCatalog struct {
	Stream              Stream     `json:"stream"`
	StreamName          string     `json:"streamName"`
	IsStreamSelected    bool       `json:"isStreamSelected"`
	SyncModes           [][]string `json:"syncModes"`
	SelectedSyncMode    []string   `json:"selectedSyncMode"`
	CursorFields        [][]string `json:"cursorFields"`
	SelectedCursorField []string   `json:"selectedCursorField"`
	PrimaryKeys         [][]string `json:"primaryKeys"`
	SelectedPrimaryKey  [][]string `json:"selectedPrimaryKey"`
}

func (f *FormFieldSpec) EnumContainsValue(value interface{}) bool {
	for _, e := range f.Enum {
		if e == value {
			return true
		}
	}
	return false
}

func (f *FormFieldSpec) DependsOnValuesIncludes(value interface{}) bool {
	for _, e := range f.DependsOnValue {
		if e == value {
			return true
		}
	}
	return false
}

func (f *FormFieldCatalog) IsSyncModeAvailable(syncMode []string) bool {
	for _, m := range f.SyncModes {
		if testEq(m, syncMode) {
			return true
		}
	}
	return false
}

func (f *FormFieldCatalog) IsCursorFieldAvailable(cursorField []string) bool {
	for _, m := range f.CursorFields {
		if testEq(m, cursorField) {
			return true
		}
	}
	return false
}

func (f *FormFieldCatalog) IsPrimaryKeyAvailable(primaryKey [][]string) bool {
	for _, p := range primaryKey {
		found := false
		for _, m := range f.PrimaryKeys {
			if testEq(m, p) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (f *Form) ToSpec() map[string]interface{} {
	result := map[string]interface{}{}

	for _, field := range f.Spec {
		if field.Ignore {
			continue
		}
		if field.Value == nil && !field.Required {
			continue
		}
		if field.DependsOnIdx != nil && !field.DependsOnValuesIncludes(f.Spec[*field.DependsOnIdx].Value) {
			continue
		}

		m := result
		for _, p := range field.Path[:len(field.Path)-1] {
			if OneOfPattern.MatchString(p) {
				continue
			}
			if m[p] == nil {
				m[p] = map[string]interface{}{}
			}
			m = m[p].(map[string]interface{})
		}
		m[field.Path[len(field.Path)-1]] = field.Value
	}

	return result
}

func (f *Form) ToConfiguredCatalog() map[string]interface{} {
	result := map[string]interface{}{
		"type": MessageTypeConfiguredCatalog,
		"configuredCatalog": map[string][]interface{}{
			"streams": nil,
		},
	}

	cc := result["configuredCatalog"].(map[string][]interface{})

	for _, field := range f.Catalog {
		if !field.IsStreamSelected {
			continue
		}

		m := map[string]interface{}{}

		m["stream"] = field.Stream

		m["sync_mode"] = field.SelectedSyncMode[0]
		if field.SelectedSyncMode[0] == SyncModeIncremental &&
			len(field.SelectedCursorField) != 0 {
			m["cursor_field"] = field.SelectedCursorField
		}

		m["destination_sync_mode"] = field.SelectedSyncMode[1]
		if (field.SelectedSyncMode[1] == DestinationSyncModeAppendDedup ||
			field.SelectedSyncMode[1] == DestinationSyncModeUpsertDedup) &&
			len(field.SelectedPrimaryKey) != 0 {
			m["primary_key"] = field.SelectedPrimaryKey
		}

		cc["streams"] = append(cc["streams"], m)
	}

	return result
}

func (f *Form) Merge(patch *Form) {
	switch f.Type {
	case FormTypeSpec:
		var isMatch func(baseField, patchField *FormFieldSpec) bool
		isMatch = func(baseField, patchField *FormFieldSpec) bool {
			if !testEq(baseField.Path, patchField.Path) {
				return false
			}
			if baseField.Type != patchField.Type {
				return false
			}
			if baseField.Multiple != patchField.Multiple {
				return false
			}
			if len(baseField.Enum) != 0 && !baseField.EnumContainsValue(patchField.Value) {
				return false
			}
			if (baseField.DependsOnIdx == nil) != (patchField.DependsOnIdx == nil) {
				return false
			}
			if baseField.DependsOnIdx != nil &&
				!isMatch(f.Spec[*baseField.DependsOnIdx], patch.Spec[*patchField.DependsOnIdx]) {
				return false
			}
			return true
		}

		for _, baseField := range f.Spec {
			for _, patchField := range patch.Spec {
				if isMatch(baseField, patchField) {
					baseField.Value = patchField.Value
					break
				}
			}
		}

	case FormTypeCatalog:
		isMatch := func(baseField, patchField *FormFieldCatalog) bool {
			if baseField.StreamName != patchField.StreamName {
				return false
			}
			if !baseField.IsSyncModeAvailable(patchField.SelectedSyncMode) {
				return false
			}
			if patchField.SelectedCursorField != nil &&
				!baseField.IsCursorFieldAvailable(patchField.SelectedCursorField) {
				return false
			}
			if patchField.SelectedPrimaryKey != nil &&
				!baseField.IsPrimaryKeyAvailable(patchField.SelectedPrimaryKey) {
				return false
			}
			return true
		}
		for _, baseField := range f.Catalog {
			for _, patchField := range patch.Catalog {
				if isMatch(baseField, patchField) {
					baseField.IsStreamSelected = patchField.IsStreamSelected
					baseField.SelectedSyncMode = patchField.SelectedSyncMode
					baseField.SelectedCursorField = patchField.SelectedCursorField
					baseField.SelectedPrimaryKey = patchField.SelectedPrimaryKey
					break
				}
			}
		}

	default:
		panic("Unhandled form type in merge")
	}
}

func testEq(a, b []string) bool {
	if (a == nil) != (b == nil) {
		return false
	}
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
