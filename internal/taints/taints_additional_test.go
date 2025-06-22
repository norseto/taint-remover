package taints

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Additional test cases for TaintKeyExists to improve coverage
func TestTaintKeyExistsAdditional(t *testing.T) {
	taints := []v1.Taint{{Key: "taint1", Effect: "NoSchedule"}, {Key: "taint2", Effect: "NoExecute"}}

	tests := []struct {
		name     string
		taints   []v1.Taint
		taintKey string
		want     bool
	}{
		{
			name:     "key exists",
			taints:   taints,
			taintKey: "taint1",
			want:     true,
		},
		{
			name:     "key does not exist",
			taints:   taints,
			taintKey: "taint3",
			want:     false,
		},
		{
			name:     "empty taints",
			taints:   []v1.Taint{},
			taintKey: "taint1",
			want:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := TaintKeyExists(test.taints, test.taintKey)
			if got != test.want {
				t.Errorf("TaintKeyExists() = %v, want %v", got, test.want)
			}
		})
	}
}

// Additional test cases for CheckTaintValidation to improve coverage
func TestCheckTaintValidationAdditional(t *testing.T) {
	tests := []struct {
		name      string
		taint     v1.Taint
		wantError bool
	}{
		{
			name: "invalid value",
			taint: v1.Taint{
				Key:    "valid-key",
				Value:  "invalid@value",
				Effect: v1.TaintEffectNoSchedule,
			},
			wantError: true,
		},
		{
			name: "empty key",
			taint: v1.Taint{
				Key:    "",
				Value:  "value",
				Effect: v1.TaintEffectNoSchedule,
			},
			wantError: true,
		},
		{
			name: "empty effect",
			taint: v1.Taint{
				Key:    "key",
				Value:  "value",
				Effect: "",
			},
			wantError: false, // Empty effect is allowed in validation
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := CheckTaintValidation(test.taint)
			gotError := err != nil
			if gotError != test.wantError {
				t.Errorf("CheckTaintValidation() gotError=%v, wantError=%v, error=%v", gotError, test.wantError, err)
			}
		})
	}
}

// Additional test cases for parseTaint to improve coverage
func TestParseTaintAdditional(t *testing.T) {
	tests := []struct {
		name          string
		taintSpec     string
		expectedTaint v1.Taint
		expectError   bool
	}{
		{
			name:        "too many parts",
			taintSpec:   "key=value:NoSchedule:extra",
			expectError: true,
		},
		{
			name:        "too many equals",
			taintSpec:   "key=value=extra:NoSchedule",
			expectError: true,
		},
		{
			name:          "key only with colon",
			taintSpec:     "key:NoSchedule",
			expectedTaint: v1.Taint{Key: "key", Value: "", Effect: v1.TaintEffectNoSchedule},
			expectError:   false,
		},
		{
			name:          "key only without colon",
			taintSpec:     "key",
			expectedTaint: v1.Taint{Key: "key", Value: "", Effect: ""},
			expectError:   false, // The function doesn't actually error on this case
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			taint, err := parseTaint(test.taintSpec)
			if test.expectError && err == nil {
				t.Errorf("parseTaint(%s) expected error, got none", test.taintSpec)
			} else if !test.expectError && err != nil {
				t.Errorf("parseTaint(%s) returned unexpected error: %v", test.taintSpec, err)
			} else if !test.expectError && !reflect.DeepEqual(taint, test.expectedTaint) {
				t.Errorf("parseTaint(%s) returned incorrect taint, got: %v, want: %v", test.taintSpec, taint, test.expectedTaint)
			}
		})
	}
}

// Additional test cases for ParseTaints to improve coverage
func TestParseTaintsAdditional(t *testing.T) {
	tests := []struct {
		name             string
		taintSpecs       []string
		expectedTaints   []v1.Taint
		expectedToRemove []v1.Taint
		expectError      bool
	}{
		{
			name:        "duplicate taints",
			taintSpecs:  []string{"key=value:NoSchedule", "key=value2:NoSchedule"},
			expectError: true,
		},
		{
			name:        "empty taint specs",
			taintSpecs:  []string{},
			expectError: false,
		},
		{
			name:        "taint without effect",
			taintSpecs:  []string{"key=value"},
			expectError: true,
		},
		{
			name:             "mixed valid and remove taints",
			taintSpecs:       []string{"key1=value1:NoSchedule", "key2-"},
			expectedTaints:   []v1.Taint{{Key: "key1", Value: "value1", Effect: v1.TaintEffectNoSchedule}},
			expectedToRemove: []v1.Taint{{Key: "key2", Effect: ""}},
			expectError:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			taints, toRemove, err := ParseTaints(test.taintSpecs)
			if test.expectError && err == nil {
				t.Errorf("ParseTaints(%v) expected error, got none", test.taintSpecs)
			} else if !test.expectError && err != nil {
				t.Errorf("ParseTaints(%v) returned unexpected error: %v", test.taintSpecs, err)
			} else if !test.expectError {
				if !reflect.DeepEqual(taints, test.expectedTaints) {
					t.Errorf("ParseTaints(%v) returned incorrect taints, got: %v, want: %v", test.taintSpecs, taints, test.expectedTaints)
				}
				if !reflect.DeepEqual(toRemove, test.expectedToRemove) {
					t.Errorf("ParseTaints(%v) returned incorrect toRemove, got: %v, want: %v", test.taintSpecs, toRemove, test.expectedToRemove)
				}
			}
		})
	}
}

// Additional test cases for RemoveTaint to improve coverage
func TestRemoveTaintAdditional(t *testing.T) {
	taint := &v1.Taint{
		Key:    "foo",
		Value:  "bar",
		Effect: "NoExecute",
	}

	t.Run("test with node having no taints", func(t *testing.T) {
		node := &v1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-no-taints"},
			Spec:       v1.NodeSpec{},
		}

		newNode, updated, err := RemoveTaint(node, taint)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if updated {
			t.Fatal("Expected the node not to be updated, but it was")
		}

		if !reflect.DeepEqual(newNode, node) {
			t.Fatalf("Expected the node to be unchanged, but got: %v", newNode)
		}
	})

	t.Run("test with nil taint", func(t *testing.T) {
		node := &v1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-nil-taint"},
			Spec: v1.NodeSpec{
				Taints: []v1.Taint{{Key: "foo", Value: "bar", Effect: "NoExecute"}},
			},
		}

		newNode, updated, err := RemoveTaint(node, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if updated {
			t.Fatal("Expected the node not to be updated, but it was")
		}
		if !reflect.DeepEqual(newNode, node) {
			t.Fatalf("Expected the node to be unchanged, but got: %v", newNode)
		}
	})
}
