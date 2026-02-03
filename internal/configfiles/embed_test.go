package configfiles

import (
	"testing"
)

// TestGetBootstrapExample tests the GetBootstrapExample function
func TestGetBootstrapExample(t *testing.T) {
	content, err := GetBootstrapExample()
	if err != nil {
		t.Fatalf("GetBootstrapExample failed: %v", err)
	}
	if len(content) == 0 {
		t.Error("GetBootstrapExample returned empty content")
	}
}

// TestGetReviewExample tests the GetReviewExample function
func TestGetReviewExample(t *testing.T) {
	content, err := GetReviewExample()
	if err != nil {
		t.Fatalf("GetReviewExample failed: %v", err)
	}
	if len(content) == 0 {
		t.Error("GetReviewExample returned empty content")
	}
}
