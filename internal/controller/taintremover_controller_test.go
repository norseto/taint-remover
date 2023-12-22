package controller

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_applyTaintRemoveOnNode(t *testing.T) {
	// Mock objects
	ctx := context.TODO()
	nodeName := "test-node"
	taintValue := "test-taint"

	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeName}, Spec: corev1.NodeSpec{Taints: []corev1.Taint{{Key: taintValue, Value: taintValue, Effect: "NoSchedule"}}}}
	nodeWithoutTaint := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: nodeName}, Spec: corev1.NodeSpec{Taints: []corev1.Taint{}}}

	testCases := []struct {
		desc     string
		node     *corev1.Node
		hasError bool
	}{
		{
			desc:     "Happy path",
			node:     node,
			hasError: false,
		},
		{
			desc:     "Node without Taint",
			node:     nodeWithoutTaint,
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cl := fake.NewClientBuilder().WithRuntimeObjects(tc.node).Build()
			r := &TaintRemoverReconciler{Client: cl}
			err := r.applyTaintRemoveOnNode(ctx, tc.node)
			if tc.hasError {
				assert.NotNil(t, err)
				assert.Equal(t, errors.IsNotFound(err), true)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
