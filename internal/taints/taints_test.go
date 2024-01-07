package taints

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestParseTaint(t *testing.T) {
	validTaint := v1.Taint{Key: "key", Value: "value", Effect: v1.TaintEffectNoSchedule}

	tests := []struct {
		name          string
		taintSpec     string
		expectedTaint v1.Taint
		expectError   bool
	}{
		{
			name:          "valid taint spec",
			taintSpec:     "key=value:NoSchedule",
			expectedTaint: validTaint,
			expectError:   false,
		},
		{
			name:        "missing effect",
			taintSpec:   "key=value",
			expectError: true,
		},
		{
			name:        "invalid effect",
			taintSpec:   "key=value:NoOp",
			expectError: true,
		},
		{
			name:        "empty key",
			taintSpec:   "=value:NoSchedule",
			expectError: true,
		},
		{
			name:        "invalid key",
			taintSpec:   "bad@key=value:NoSchedule",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			taint, err := parseTaint(test.taintSpec)
			if test.expectError && err == nil {
				t.Errorf("parseTaint(%s) expected error, got none", test.taintSpec)
			} else if !test.expectError && err != nil {
				t.Errorf("parseTaint(%s) returned unexpected error: %v", test.taintSpec, err)
			} else if !reflect.DeepEqual(taint, test.expectedTaint) {
				t.Errorf("parseTaint(%s) returned incorrect taint, got: %v, want: %v", test.taintSpec, taint, test.expectedTaint)
			}
		})
	}
}

func TestParseTaints(t *testing.T) {
	validTaint1 := v1.Taint{Key: "key1", Value: "value1", Effect: v1.TaintEffectNoSchedule}
	validTaint2 := v1.Taint{Key: "key2", Value: "", Effect: v1.TaintEffectNoExecute}

	tests := []struct {
		name             string
		taintSpecs       []string
		expectedTaints   []v1.Taint
		expectedToRemove []v1.Taint
		expectError      bool
	}{
		{
			name:           "valid taint specs",
			taintSpecs:     []string{"key1=value1:NoSchedule", "key2:NoExecute"},
			expectedTaints: []v1.Taint{validTaint1, validTaint2},
		},
		{
			name:             "valid remove taint",
			taintSpecs:       []string{"key1-"},
			expectedToRemove: []v1.Taint{{Key: "key1", Effect: ""}},
		},
		{
			name:        "invalid taint spec",
			taintSpecs:  []string{"bad@key=value:NoSchedule"},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			taints, toRemove, err := ParseTaints(test.taintSpecs)
			if test.expectError && err == nil {
				t.Errorf("ParseTaints(%v) expected error, got none", test.taintSpecs)
			} else if !test.expectError && err != nil {
				t.Errorf("ParseTaints(%v) returned unexpected error: %v", test.taintSpecs, err)
			} else if !reflect.DeepEqual(taints, test.expectedTaints) {
				t.Errorf("ParseTaints(%v) returned incorrect taints, got: %v, want: %v", test.taintSpecs, taints, test.expectedTaints)
			} else if !reflect.DeepEqual(toRemove, test.expectedToRemove) {
				t.Errorf("ParseTaints(%v) returned incorrect toRemove, got: %v, want: %v", test.taintSpecs, toRemove, test.expectedToRemove)
			}
		})
	}
}

func TestCheckIfTaintsAlreadyExists(t *testing.T) {
	existingTaint1 := v1.Taint{Key: "taint1", Effect: v1.TaintEffectNoSchedule}
	existingTaint2 := v1.Taint{Key: "taint2", Effect: v1.TaintEffectNoExecute}
	newTaint1 := v1.Taint{Key: "taint1", Effect: v1.TaintEffectNoSchedule}
	newTaint2 := v1.Taint{Key: "taint3", Effect: v1.TaintEffectNoExecute}

	tests := []struct {
		name      string
		oldTaints []v1.Taint
		newTaints []v1.Taint
		want      string
	}{
		{
			name:      "one duplicate taint",
			oldTaints: []v1.Taint{existingTaint1, existingTaint2},
			newTaints: []v1.Taint{newTaint1, newTaint2},
			want:      "taint1",
		},
		{
			name:      "no duplicate taints",
			oldTaints: []v1.Taint{existingTaint1},
			newTaints: []v1.Taint{newTaint2},
			want:      "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := CheckIfTaintsAlreadyExists(test.oldTaints, test.newTaints)
			if got != test.want {
				t.Errorf("CheckIfTaintsAlreadyExists() = %v, want %v", got, test.want)
			}
		})
	}
}

// Add test cases for other functions

func TestDeleteTaintsByKey(t *testing.T) {
	taints := []v1.Taint{{Key: "taint1"}, {Key: "taint2"}}
	taintKey := "taint1"
	wantTaints := []v1.Taint{{Key: "taint2"}}
	wantDeleted := true

	gotTaints, gotDeleted := DeleteTaintsByKey(taints, taintKey)

	if !reflect.DeepEqual(gotTaints, wantTaints) {
		t.Errorf("DeleteTaintsByKey() gotTaints = %v, want %v", gotTaints, wantTaints)
	}
	if gotDeleted != wantDeleted {
		t.Errorf("DeleteTaintsByKey() gotDeleted = %v, want %v", gotDeleted, wantDeleted)
	}
}

func TestDeleteTaint(t *testing.T) {
	taints := []v1.Taint{{Key: "taint1", Effect: "NoSchedule"}, {Key: "taint2", Effect: "NoExecute"}}
	taint := v1.Taint{Key: "taint1", Effect: "NoSchedule"}
	wantTaints := []v1.Taint{{Key: "taint2", Effect: "NoExecute"}}
	wantDeleted := true

	gotTaints, gotDeleted := DeleteTaint(taints, &taint)

	if !reflect.DeepEqual(gotTaints, wantTaints) {
		t.Errorf("DeleteTaint() gotTaints = %v, want %v", gotTaints, wantTaints)
	}
	if gotDeleted != wantDeleted {
		t.Errorf("DeleteTaint() gotDeleted = %v, want %v", gotDeleted, wantDeleted)
	}
}

func TestRemoveTaint(t *testing.T) {
	taint := &v1.Taint{
		Key:    "foo",
		Value:  "bar",
		Effect: "NoExecute",
	}

	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "node1"},
		Spec: v1.NodeSpec{
			Taints: []v1.Taint{*taint},
		},
	}

	t.Run("test with taint exist", func(t *testing.T) {
		newNode, updated, err := RemoveTaint(node, taint)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if !updated {
			t.Fatal("Expected node to be updated, but it was not")
		}

		if len(newNode.Spec.Taints) != 0 {
			t.Fatalf("Expected taint to be removed, but it's still there: %v", newNode.Spec.Taints)
		}
	})

	t.Run("test with taint not exist", func(t *testing.T) {
		newNode, updated, err := RemoveTaint(node, &v1.Taint{Key: "foo1", Value: "bar", Effect: "NoExecute"})

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
func TestTaintExists(t *testing.T) {
	taints := []v1.Taint{{Key: "taint1", Effect: "NoSchedule"}, {Key: "taint2", Effect: "NoExecute"}}
	taint1 := v1.Taint{Key: "taint1", Effect: "NoSchedule"}
	taint3 := v1.Taint{Key: "taint3", Effect: "NoSchedule"}

	tests := []struct {
		name        string
		taints      []v1.Taint
		taintToFind v1.Taint
		want        bool
	}{
		{
			name:        "taint exists",
			taints:      taints,
			taintToFind: taint1,
			want:        true,
		},
		{
			name:        "taint does not exist",
			taints:      taints,
			taintToFind: taint3,
			want:        false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := TaintExists(test.taints, &test.taintToFind)
			if got != test.want {
				t.Errorf("TaintExists() = %v, want %v", got, test.want)
			}
		})
	}
}

// Add test cases for other functions

func TestTaintKeyExists(t *testing.T) {
	taints := []v1.Taint{{Key: "taint1", Effect: "NoSchedule"}, {Key: "taint2", Effect: "NoExecute"}}
	taintKey := "taint1"

	got := TaintKeyExists(taints, taintKey)
	want := true

	if got != want {
		t.Errorf("TaintKeyExists() = %v, want %v", got, want)
	}
}

func TestTaintSetDiff(t *testing.T) {
	taintsOld := []v1.Taint{{Key: "taint1", Effect: "NoSchedule"}, {Key: "taint2", Effect: "NoExecute"}}
	taintsNew := []v1.Taint{{Key: "taint2", Effect: "NoExecute"}, {Key: "taint3", Effect: "NoSchedule"}}

	wantToAdd := []*v1.Taint{{Key: "taint3", Effect: "NoSchedule"}}
	wantToRemove := []*v1.Taint{{Key: "taint1", Effect: "NoSchedule"}}

	gotToAdd, gotToRemove := TaintSetDiff(taintsNew, taintsOld)

	if !reflect.DeepEqual(gotToAdd, wantToAdd) {
		t.Errorf("TaintSetDiff() gotToAdd = %v, want %v", gotToAdd, wantToAdd)
	}

	if !reflect.DeepEqual(gotToRemove, wantToRemove) {
		t.Errorf("TaintSetDiff() gotToRemove = %v, want %v", gotToRemove, wantToRemove)
	}
}

func TestTaintSetFilter(t *testing.T) {
	taints := []v1.Taint{
		{Key: "taint1", Effect: "NoSchedule"},
		{Key: "taint2", Effect: "NoExecute"},
	}

	fn := func(taint *v1.Taint) bool {
		return taint.Effect == "NoSchedule"
	}

	want := []v1.Taint{{Key: "taint1", Effect: "NoSchedule"}}

	got := TaintSetFilter(taints, fn)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("TaintSetFilter() = %v, want %v", got, want)
	}
}

func TestCheckTaintValidation(t *testing.T) {
	validTaint := v1.Taint{Key: "taint1", Value: "value1", Effect: v1.TaintEffectNoSchedule}
	invalidKeyTaint := v1.Taint{Key: "bad@key", Value: "value2", Effect: v1.TaintEffectNoExecute}
	invalidEffectTaint := v1.Taint{Key: "taint2", Value: "value3", Effect: v1.TaintEffect("InvalidEffect")}

	tests := []struct {
		name      string
		taint     v1.Taint
		wantError bool
	}{
		{
			name:      "valid taint",
			taint:     validTaint,
			wantError: false,
		},
		{
			name:      "invalid key",
			taint:     invalidKeyTaint,
			wantError: true,
		},
		{
			name:      "invalid effect",
			taint:     invalidEffectTaint,
			wantError: true,
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
