package v1alpha1

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAddToSchemeRegistersTypes(t *testing.T) {
	scheme := runtime.NewScheme()

	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme returned error: %v", err)
	}

	if !scheme.IsVersionRegistered(GroupVersion) {
		t.Fatalf("GroupVersion %s not registered", GroupVersion)
	}

	obj := &TaintRemover{}
	gvks, _, err := scheme.ObjectKinds(obj)
	if err != nil {
		t.Fatalf("ObjectKinds returned error: %v", err)
	}
	if len(gvks) == 0 {
		t.Fatalf("expected at least one GVK for TaintRemover")
	}
	if gvks[0].Group != GroupVersion.Group || gvks[0].Version != GroupVersion.Version {
		t.Fatalf("unexpected GVK registered: got %s/%s", gvks[0].Group, gvks[0].Version)
	}
}

func TestDeepCopyFunctions(t *testing.T) {
	original := &TaintRemover{
		Spec: TaintRemoverSpec{
			Taints: []corev1.Taint{{Key: "key", Value: "value", Effect: corev1.TaintEffectNoSchedule}},
		},
	}

	copied := original.DeepCopy()
	if copied == original {
		t.Fatalf("DeepCopy should allocate a new instance")
	}
	if copied.Spec.Taints[0].Key != original.Spec.Taints[0].Key {
		t.Fatalf("DeepCopy should retain field values")
	}

	var into TaintRemover
	original.DeepCopyInto(&into)
	if into.Spec.Taints == nil {
		t.Fatalf("DeepCopyInto should populate target")
	}
	into.Spec.Taints[0].Value = "mutated"
	if original.Spec.Taints[0].Value == "mutated" {
		t.Fatalf("DeepCopyInto should perform deep copy of slices")
	}

	list := &TaintRemoverList{Items: []TaintRemover{*original}}
	listCopy := list.DeepCopy()
	if listCopy == list {
		t.Fatalf("DeepCopy on list should allocate a new instance")
	}
	if len(listCopy.Items) != len(list.Items) {
		t.Fatalf("DeepCopy on list should retain item count")
	}

	// Ensure DeepCopyObject returns runtime.Object implementations.
	if obj := original.DeepCopyObject(); obj == nil {
		t.Fatalf("DeepCopyObject should not return nil for non-nil receiver")
	}
	if obj := list.DeepCopyObject(); obj == nil {
		t.Fatalf("DeepCopyObject should not return nil for list")
	}

	nilList := (*TaintRemoverList)(nil)
	if obj := nilList.DeepCopyObject(); obj != nil {
		t.Fatalf("expected DeepCopyObject on nil list to return nil")
	}

	nilRemover := (*TaintRemover)(nil)
	if obj := nilRemover.DeepCopyObject(); obj != nil {
		t.Fatalf("expected DeepCopyObject on nil remover to return nil")
	}

	specCopy := original.Spec.DeepCopy()
	if specCopy == nil {
		t.Fatalf("expected DeepCopy on spec to return value")
	}
	specCopy.Taints[0].Value = "copied"
	if original.Spec.Taints[0].Value == "copied" {
		t.Fatalf("DeepCopy on spec should deep copy taints slice")
	}

	var nilSpec *TaintRemoverSpec
	if nilSpec.DeepCopy() != nil {
		t.Fatalf("expected nil spec DeepCopy to return nil")
	}

	status := &TaintRemoverStatus{}
	if status.DeepCopy() == nil {
		t.Fatalf("expected DeepCopy on status to return value")
	}

	var nilStatus *TaintRemoverStatus
	if nilStatus.DeepCopy() != nil {
		t.Fatalf("expected nil status DeepCopy to return nil")
	}
}
