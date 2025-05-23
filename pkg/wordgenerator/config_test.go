package wordgenerator

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFileLayoutDetail_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		expected      FileLayoutDetail
		expectedError bool
		checkParsedRotate func(t *testing.T, fld *FileLayoutDetail) // For checking the internal parsedRotate
	}{
		{
			name:     "simple layout",
			jsonData: `{"files": ["a.jpg"], "layout": "full_page"}`,
			expected: FileLayoutDetail{
				Files:  []string{"a.jpg"},
				Layout: "full_page",
			},
		},
		{
			name:     "rotate as single integer",
			jsonData: `{"files": ["a.jpg"], "layout": "full_page", "rotate": 90}`,
			expected: FileLayoutDetail{
				Files:  []string{"a.jpg"},
				Layout: "full_page",
				Rotate: json.RawMessage(`90`),
			},
			checkParsedRotate: func(t *testing.T, fld *FileLayoutDetail) {
				if len(fld.parsedRotate) != 1 || fld.parsedRotate[0].(float64) != 90 {
					t.Errorf("parsedRotate unexpected: got %v, want [90]", fld.parsedRotate)
				}
				rot := fld.getRotationForIndex(0)
				if rot == nil || *rot != 90 {
					t.Errorf("getRotationForIndex(0) unexpected: got %v, want 90", rot)
				}
			},
		},
		{
			name:     "rotate as array of integers",
			jsonData: `{"files": ["a.jpg", "b.jpg"], "layout": "full_page", "rotate": [90, 180]}`,
			expected: FileLayoutDetail{
				Files:  []string{"a.jpg", "b.jpg"},
				Layout: "full_page",
				Rotate: json.RawMessage(`[90, 180]`),
			},
			checkParsedRotate: func(t *testing.T, fld *FileLayoutDetail) {
				expectedPR := []interface{}{float64(90), float64(180)}
				if !reflect.DeepEqual(fld.parsedRotate, expectedPR) {
					t.Errorf("parsedRotate unexpected: got %v, want %v", fld.parsedRotate, expectedPR)
				}
				rot0 := fld.getRotationForIndex(0)
				if rot0 == nil || *rot0 != 90 {
					t.Errorf("getRotationForIndex(0) unexpected: got %v, want 90", rot0)
				}
				rot1 := fld.getRotationForIndex(1)
				if rot1 == nil || *rot1 != 180 {
					t.Errorf("getRotationForIndex(1) unexpected: got %v, want 180", rot1)
				}
				if fld.getRotationForIndex(2) != nil {
					t.Errorf("getRotationForIndex(2) unexpected: got non-nil, want nil")
				}
			},
		},
		{
			name:     "rotate as array with null",
			jsonData: `{"files": ["a.jpg", "b.jpg"], "layout": "full_page", "rotate": [90, null]}`,
			expected: FileLayoutDetail{
				Files:  []string{"a.jpg", "b.jpg"},
				Layout: "full_page",
				Rotate: json.RawMessage(`[90, null]`),
			},
			checkParsedRotate: func(t *testing.T, fld *FileLayoutDetail) {
				expectedPR := []interface{}{float64(90), nil}
				if !reflect.DeepEqual(fld.parsedRotate, expectedPR) {
					t.Errorf("parsedRotate unexpected: got %v, want %v", fld.parsedRotate, expectedPR)
				}
				rot0 := fld.getRotationForIndex(0)
				if rot0 == nil || *rot0 != 90 {
					t.Errorf("getRotationForIndex(0) unexpected: got %v, want 90", rot0)
				}
				if fld.getRotationForIndex(1) != nil {
					t.Errorf("getRotationForIndex(1) unexpected: got non-nil, want nil")
				}
			},
		},
		{
			name:     "custom ratio layout",
			jsonData: `{"files": ["a.jpg", "b.jpg"], "layout": "custom_ratio", "ratios": [0.6, 0.4], "orientation": "vertical", "ratio_orientation": "vertical"}`,
			expected: FileLayoutDetail{
				Files:            []string{"a.jpg", "b.jpg"},
				Layout:           "custom_ratio",
				Ratios:           []float64{0.6, 0.4},
				Orientation:      "vertical",
				RatioOrientation: "vertical",
			},
		},
		{
			name:          "malformed json",
			jsonData:      `{"files": ["a.jpg"], "layout": "full_page",`,
			expectedError: true,
		},
		{
			name:          "missing files field",
			jsonData:      `{"layout": "full_page"}`,
			expectedError: false, // Files is not strictly required by JSON schema, but logic might fail later
			expected: FileLayoutDetail{
				Layout: "full_page", // Files will be nil
			},
		},
		{
			name:          "incorrect data type for files",
			jsonData:      `{"files": "a.jpg", "layout": "full_page"}`,
			expectedError: true,
		},
		{
			name:          "incorrect data type for ratios",
			jsonData:      `{"files": ["a.jpg"], "layout": "custom_ratio", "ratios": "0.5,0.5"}`,
			expectedError: true,
		},
		{
            name:     "rotate with single null in array",
            jsonData: `{"files": ["a.jpg"], "layout": "full_page", "rotate": [null]}`,
            expected: FileLayoutDetail{
                Files:  []string{"a.jpg"},
                Layout: "full_page",
                Rotate: json.RawMessage(`[null]`),
            },
            checkParsedRotate: func(t *testing.T, fld *FileLayoutDetail) {
                expectedPR := []interface{}{nil}
                if !reflect.DeepEqual(fld.parsedRotate, expectedPR) {
                    t.Errorf("parsedRotate unexpected: got %v, want %v", fld.parsedRotate, expectedPR)
                }
                if fld.getRotationForIndex(0) != nil {
                    t.Errorf("getRotationForIndex(0) unexpected: got non-nil, want nil")
                }
            },
        },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actual FileLayoutDetail
			err := json.Unmarshal([]byte(tt.jsonData), &actual)

			if (err != nil) != tt.expectedError {
				t.Errorf("UnmarshalJSON() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if tt.expectedError {
				return // Don't compare further if an error was expected and occurred
			}

			// Compare fields except Rotate and parsedRotate
			if actual.Layout != tt.expected.Layout ||
				!reflect.DeepEqual(actual.Files, tt.expected.Files) ||
				!reflect.DeepEqual(actual.Ratios, tt.expected.Ratios) ||
				actual.Orientation != tt.expected.Orientation ||
				actual.RatioOrientation != tt.expected.RatioOrientation {
				t.Errorf("UnmarshalJSON() got = %+v, want (excluding Rotate, parsedRotate) = %+v", actual, tt.expected)
			}

			// Compare raw Rotate field
			if string(actual.Rotate) != string(tt.expected.Rotate) {
				t.Errorf("UnmarshalJSON() raw Rotate field got = %s, want = %s", string(actual.Rotate), string(tt.expected.Rotate))
			}
			
			// Check parsedRotate and getRotationForIndex logic if a checker is provided
			if tt.checkParsedRotate != nil {
				tt.checkParsedRotate(t, &actual)
			}
		})
	}
}

func TestDocxConfig_UnmarshalJSON(t *testing.T) {
	jsonData := `
{
  "doc1.docx": {
    "files": ["image1.jpg", "image2.png"],
    "layout": "vertical_2_split",
    "rotate": [90, null]
  },
  "doc2.docx": {
    "files": ["main.tif"],
    "layout": "full_page",
    "rotate": 180
  }
}
`
	var config DocxConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal DocxConfig: %v", err)
	}

	if len(config) != 2 {
		t.Fatalf("Expected 2 document configurations, got %d", len(config))
	}

	// Check doc1.docx
	doc1, ok := config["doc1.docx"]
	if !ok {
		t.Fatal("doc1.docx not found in parsed config")
	}
	if doc1.Layout != "vertical_2_split" {
		t.Errorf("doc1.layout: got %s, want vertical_2_split", doc1.Layout)
	}
	if len(doc1.Files) != 2 || doc1.Files[0] != "image1.jpg" || doc1.Files[1] != "image2.png" {
		t.Errorf("doc1.files: got %v, want [image1.jpg, image2.png]", doc1.Files)
	}
	expectedDoc1ParsedRotate := []interface{}{float64(90), nil}
	if !reflect.DeepEqual(doc1.parsedRotate, expectedDoc1ParsedRotate) {
		t.Errorf("doc1.parsedRotate: got %v, want %v", doc1.parsedRotate, expectedDoc1ParsedRotate)
	}
	rot0_doc1 := doc1.getRotationForIndex(0)
	if rot0_doc1 == nil || *rot0_doc1 != 90 {
		t.Errorf("doc1.getRotationForIndex(0) unexpected: got %v, want 90", rot0_doc1)
	}
	if doc1.getRotationForIndex(1) != nil {
		t.Errorf("doc1.getRotationForIndex(1) unexpected: got non-nil, want nil")
	}


	// Check doc2.docx
	doc2, ok := config["doc2.docx"]
	if !ok {
		t.Fatal("doc2.docx not found in parsed config")
	}
	if doc2.Layout != "full_page" {
		t.Errorf("doc2.layout: got %s, want full_page", doc2.Layout)
	}
	if len(doc2.Files) != 1 || doc2.Files[0] != "main.tif" {
		t.Errorf("doc2.files: got %v, want [main.tif]", doc2.Files)
	}
	expectedDoc2ParsedRotate := []interface{}{float64(180)}
	if !reflect.DeepEqual(doc2.parsedRotate, expectedDoc2ParsedRotate) {
		t.Errorf("doc2.parsedRotate: got %v, want %v", doc2.parsedRotate, expectedDoc2ParsedRotate)
	}
	rot0_doc2 := doc2.getRotationForIndex(0)
	if rot0_doc2 == nil || *rot0_doc2 != 180 {
		t.Errorf("doc2.getRotationForIndex(0) unexpected: got %v, want 180", rot0_doc2)
	}
}

func TestGetRotationForIndex_EdgeCases(t *testing.T) {
	detail := FileLayoutDetail{}

	// No rotation specified
	if detail.getRotationForIndex(0) != nil {
		t.Error("Expected nil for no rotation, got a value")
	}

	// Single rotation, index out of bounds (should still apply)
	detail.parsedRotate = []interface{}{float64(45)}
	rot := detail.getRotationForIndex(5) // Index 5, but only one global rotation
	if rot == nil || *rot != 45 {
		t.Errorf("Expected 45 for single rotation, index OOB, got %v", rot)
	}
	
	// Array rotation, index out of bounds
	detail.parsedRotate = []interface{}{float64(90), float64(180)}
	if detail.getRotationForIndex(2) != nil {
		t.Error("Expected nil for array rotation, index OOB, got a value")
	}

	// Array rotation, nil entry
	detail.parsedRotate = []interface{}{nil, float64(90)}
	if detail.getRotationForIndex(0) != nil {
		t.Error("Expected nil for nil entry in array rotation, got a value")
	}
	rot = detail.getRotationForIndex(1)
	if rot == nil || *rot != 90 {
		t.Errorf("Expected 90 for second entry in array rotation, got %v", rot)
	}

	// Array rotation, non-float64 entry (should be handled by UnmarshalJSON, but test robustness)
	detail.parsedRotate = []interface{}{"not a number"}
	if detail.getRotationForIndex(0) != nil {
		t.Error("Expected nil for invalid type in parsedRotate, got a value")
	}
}
