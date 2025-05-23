package wordgenerator

import (
	"encoding/json"
)

// DocxConfig represents the overall JSON configuration structure.
// It's a map where keys are output DOCX filenames.
type DocxConfig map[string]FileLayoutDetail

// FileLayoutDetail defines the layout and files for a single DOCX.
type FileLayoutDetail struct {
	Files             []string        `json:"files"`
	Layout            string          `json:"layout"`
	Rotate            json.RawMessage `json:"rotate,omitempty"` // Can be int or []interface{} (mixed int/nil)
	Ratios            []float64       `json:"ratios,omitempty"`
	Orientation       string          `json:"orientation,omitempty"`
	RatioOrientation  string          `json:"ratio_orientation,omitempty"`
	parsedRotate      []interface{}   // Internal field to store parsed rotation values
}

// ParsedRotateValue represents a rotation value for a single image.
// It can be an integer (degrees) or nil (no rotation).
type ParsedRotateValue struct {
	Degrees *int // Pointer to allow nil for no rotation
}

// getRotationForIndex returns the rotation degrees for a specific image index.
// It handles cases where 'rotate' is a single value (applies to all) or an array.
// If no rotation is specified for an index or if rotation is nil, returns nil.
func (d *FileLayoutDetail) getRotationForIndex(index int) *int {
	if len(d.parsedRotate) == 0 {
		return nil // No rotation specified at all
	}

	if len(d.parsedRotate) == 1 { // Single rotation value applies to all
		if d.parsedRotate[0] == nil {
			return nil
		}
		if val, ok := d.parsedRotate[0].(float64); ok { // JSON numbers are float64
			deg := int(val)
			return &deg
		}
		return nil // Should not happen if parsing is correct
	}

	if index < len(d.parsedRotate) {
		if d.parsedRotate[index] == nil {
			return nil
		}
		if val, ok := d.parsedRotate[index].(float64); ok { // JSON numbers are float64
			deg := int(val)
			return &deg
		}
		return nil // Should not happen
	}
	return nil // No specific rotation for this index
}

// UnmarshalJSON custom unmarshaller for FileLayoutDetail to handle flexible 'rotate' field.
func (d *FileLayoutDetail) UnmarshalJSON(data []byte) error {
	type Alias FileLayoutDetail // Create an alias to avoid recursion
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(d),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if len(d.Rotate) > 0 {
		// Try to unmarshal 'rotate' as a single int (actually float64 from JSON)
		var singleRotate float64
		if err := json.Unmarshal(d.Rotate, &singleRotate); err == nil {
			d.parsedRotate = []interface{}{singleRotate}
			return nil
		}

		// Try to unmarshal 'rotate' as an array of interfaces (to handle potential nils)
		var arrayRotate []interface{}
		if err := json.Unmarshal(d.Rotate, &arrayRotate); err == nil {
			d.parsedRotate = arrayRotate
			return nil
		}
		// If it's neither, it might be an error or a type not handled,
		// but we'll let it pass and rely on getRotationForIndex to manage.
		// Or, you could return an error here if strict parsing is required:
		// return fmt.Errorf("failed to parse 'rotate' field: %s", string(d.Rotate))
	}
	return nil
}
