package controller

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nodesv1alpha1 "github.com/norseto/taint-remover/api/v1alpha1"
)

var unitTestScheme = runtime.NewScheme()

func init() {
	_ = corev1.AddToScheme(unitTestScheme)
	_ = nodesv1alpha1.AddToScheme(unitTestScheme)
}

type erroringClient struct {
	client.Client
	listErr  error
	getErr   error
	patchErr error
	listHook func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
}

func (c *erroringClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if c.listHook != nil {
		if err := c.listHook(ctx, list, opts...); err != nil {
			return err
		}
	}
	if c.listErr != nil {
		return c.listErr
	}
	return c.Client.List(ctx, list, opts...)
}

func (c *erroringClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	if c.getErr != nil {
		return c.getErr
	}
	return c.Client.Get(ctx, key, obj, opts...)
}

func (c *erroringClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if c.patchErr != nil {
		return c.patchErr
	}
	return c.Client.Patch(ctx, obj, patch, opts...)
}

func TestReconcileReturnsErrorWhenListingTaintsFails(t *testing.T) {
	baseClient := fake.NewClientBuilder().WithScheme(unitTestScheme).Build()
	c := &erroringClient{Client: baseClient, listErr: errors.New("list failure")}

	reconciler := &TaintRemoverReconciler{Client: c}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{}); err == nil {
		t.Fatalf("expected error when listing taint removers fails")
	}
}

func TestReconcileReturnsWithoutTaints(t *testing.T) {
	reconciler := &TaintRemoverReconciler{Client: fake.NewClientBuilder().WithScheme(unitTestScheme).Build()}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{}); err != nil {
		t.Fatalf("did not expect error with no taints: %v", err)
	}
}

func TestReconcileHandlesNodeListError(t *testing.T) {
	remover := &nodesv1alpha1.TaintRemover{
		ObjectMeta: metav1.ObjectMeta{Name: "r"},
		Spec:       nodesv1alpha1.TaintRemoverSpec{Taints: []corev1.Taint{{Key: "k", Effect: corev1.TaintEffectNoSchedule}}},
	}
	baseClient := fake.NewClientBuilder().WithScheme(unitTestScheme).WithObjects(remover).Build()
	c := &erroringClient{Client: baseClient, listHook: func(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
		switch list.(type) {
		case *corev1.NodeList:
			return errors.New("node list failure")
		default:
			return nil
		}
	}}

	reconciler := &TaintRemoverReconciler{Client: c}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{}); err == nil {
		t.Fatalf("expected error when node listing fails")
	}
}

func TestReconcilePatchError(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "p"},
		Spec:       corev1.NodeSpec{Taints: []corev1.Taint{{Key: "k", Effect: corev1.TaintEffectNoSchedule}}},
	}
	remover := &nodesv1alpha1.TaintRemover{
		ObjectMeta: metav1.ObjectMeta{Name: "p"},
		Spec:       nodesv1alpha1.TaintRemoverSpec{Taints: []corev1.Taint{{Key: "k", Effect: corev1.TaintEffectNoSchedule}}},
	}
	baseClient := fake.NewClientBuilder().WithScheme(unitTestScheme).WithObjects(node.DeepCopy(), remover).Build()
	c := &erroringClient{Client: baseClient, patchErr: errors.New("patch failure")}

	reconciler := &TaintRemoverReconciler{Client: c}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{}); err == nil {
		t.Fatalf("expected error when patching nodes fails")
	}
}

func TestApplyRemoveTaintOnNodePropagatesGetError(t *testing.T) {
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node"}}
	baseClient := fake.NewClientBuilder().WithScheme(unitTestScheme).Build()
	c := &erroringClient{Client: baseClient, getErr: errors.New("get failure")}

	if err := applyRemoveTaintOnNode(context.Background(), c, node); err == nil {
		t.Fatalf("expected error when node retrieval fails")
	}
}

func TestApplyRemoveTaintOnNodePatchError(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "patch"},
		Spec:       corev1.NodeSpec{Taints: []corev1.Taint{{Key: "k", Effect: corev1.TaintEffectNoSchedule}}},
	}
	remover := &nodesv1alpha1.TaintRemover{
		ObjectMeta: metav1.ObjectMeta{Name: "patch"},
		Spec:       nodesv1alpha1.TaintRemoverSpec{Taints: []corev1.Taint{{Key: "k", Effect: corev1.TaintEffectNoSchedule}}},
	}
	baseClient := fake.NewClientBuilder().WithScheme(unitTestScheme).WithObjects(node.DeepCopy(), remover).Build()
	c := &erroringClient{Client: baseClient, patchErr: errors.New("patch failure")}

	if err := applyRemoveTaintOnNode(context.Background(), c, node); err == nil {
		t.Fatalf("expected error when patching node fails")
	}
}

func TestMakeNewTaintsForNodeHandlesNilAndRemovals(t *testing.T) {
	if taints, removed := makeNewTaintsForNode(nil, nil); taints != nil || removed {
		t.Fatalf("expected nil and false for nil target")
	}

	node := &corev1.Node{
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{{Key: "k", Value: "v", Effect: corev1.TaintEffectNoSchedule}},
		},
	}
	target := []*corev1.Taint{{Key: "k", Value: "v", Effect: corev1.TaintEffectNoSchedule}}
	taints, removed := makeNewTaintsForNode(node, target)
	if !removed {
		t.Fatalf("expected taint removal to be reported")
	}
	if len(taints) != 0 {
		t.Fatalf("expected all taints removed, got %d", len(taints))
	}
}

func TestMakeNewTaintsForNodeNoMatches(t *testing.T) {
	node := &corev1.Node{
		Spec: corev1.NodeSpec{Taints: []corev1.Taint{{Key: "other", Effect: corev1.TaintEffectNoSchedule}}},
	}
	result, removed := makeNewTaintsForNode(node, []*corev1.Taint{{Key: "k", Effect: corev1.TaintEffectNoSchedule}})
	if removed {
		t.Fatalf("expected no removals when taint does not exist")
	}
	if len(result) != 1 {
		t.Fatalf("expected taints to remain unchanged")
	}
}

func TestPatchNodesToRemoveTaintsContinuesOnError(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "n1"},
		Spec:       corev1.NodeSpec{Taints: []corev1.Taint{{Key: "k", Effect: corev1.TaintEffectNoSchedule}}},
	}
	baseClient := fake.NewClientBuilder().WithScheme(unitTestScheme).WithObjects(node.DeepCopy()).Build()
	c := &erroringClient{Client: baseClient, patchErr: errors.New("patch failure")}

	removed, err := patchNodesToRemoveTaints(context.Background(), c, []*corev1.Node{node}, []*corev1.Taint{{Key: "k", Effect: corev1.TaintEffectNoSchedule}})
	if err == nil {
		t.Fatalf("expected error to be returned when patching fails")
	}
	if removed != 0 {
		t.Fatalf("expected no nodes patched successfully, got %d", removed)
	}
}

func TestGetAllTaintsToRemoveDeduplicates(t *testing.T) {
	remover := &nodesv1alpha1.TaintRemover{
		ObjectMeta: metav1.ObjectMeta{Name: "tr"},
		Spec: nodesv1alpha1.TaintRemoverSpec{
			Taints: []corev1.Taint{
				{Key: "k", Effect: corev1.TaintEffectNoSchedule},
				{Key: "k", Effect: corev1.TaintEffectNoSchedule},
			},
		},
	}
	baseClient := fake.NewClientBuilder().WithScheme(unitTestScheme).WithObjects(remover).Build()

	taints, err := getAllTaintsToRemove(context.Background(), baseClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(taints) != 1 {
		t.Fatalf("expected duplicate taints to be coalesced, got %d", len(taints))
	}
}

func TestGetTaintedNodesFiltersAndErrors(t *testing.T) {
	nodeWith := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "with"}, Spec: corev1.NodeSpec{Taints: []corev1.Taint{{Key: "k", Effect: corev1.TaintEffectNoSchedule}}}}
	nodeWithout := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "without"}}
	baseClient := fake.NewClientBuilder().WithScheme(unitTestScheme).WithObjects(nodeWith, nodeWithout).Build()

	nodes, err := getTaintedNodes(context.Background(), baseClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Name != "with" {
		t.Fatalf("expected only tainted node to be returned")
	}

	failingClient := &erroringClient{Client: baseClient, listErr: errors.New("list failure")}
	if _, err := getTaintedNodes(context.Background(), failingClient); err == nil {
		t.Fatalf("expected error when listing nodes fails")
	}
}

func TestNodeHandlerNoOps(t *testing.T) {
	handler := &nodeHandler{r: &TaintRemoverReconciler{Client: fake.NewClientBuilder().WithScheme(unitTestScheme).Build()}}
	ctx := context.Background()
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "noop"}}

	var queue workqueue.TypedRateLimitingInterface[reconcile.Request]
	handler.Delete(ctx, event.TypedDeleteEvent[client.Object]{Object: node}, queue)
	handler.Generic(ctx, event.TypedGenericEvent[client.Object]{Object: node}, queue)
}

func TestNodeHandlerCreateAndUpdate(t *testing.T) {
	fakeClient := &erroringClient{Client: fake.NewClientBuilder().WithScheme(unitTestScheme).Build(), getErr: errors.New("fetch failure")}
	handler := &nodeHandler{r: &TaintRemoverReconciler{Client: fakeClient}}
	ctx := context.Background()
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "event"}}

	var queue workqueue.TypedRateLimitingInterface[reconcile.Request]
	handler.Create(ctx, event.TypedCreateEvent[client.Object]{Object: node}, queue)
	handler.Update(ctx, event.TypedUpdateEvent[client.Object]{ObjectNew: node}, queue)
}

func TestPatchNodeReturnsPatchErrors(t *testing.T) {
	ctx := context.Background()
	client := &erroringClient{Client: fake.NewClientBuilder().WithScheme(unitTestScheme).Build(), patchErr: errors.New("patch failure")}

	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "marshal"}}
	patch := &nodeSpecPatchPayload{Spec: nodeSpecPatchSpec{}}

	if err := patchNode(ctx, client, node, patch); err == nil {
		t.Fatalf("expected patch error to be returned")
	}
}

func TestGetNodeAndCheckTaintsNotFound(t *testing.T) {
	baseClient := fake.NewClientBuilder().WithScheme(unitTestScheme).Build()
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "missing"}}

	found, err := getNodeAndCheckTaints(context.Background(), baseClient, node)
	if err == nil {
		t.Fatalf("expected not found error")
	}
	if found != nil {
		t.Fatalf("expected no node to be returned")
	}
}

func TestApplyRemoveTaintOnNodeNoTaints(t *testing.T) {
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "without"}}
	baseClient := fake.NewClientBuilder().WithScheme(unitTestScheme).WithObjects(node).Build()

	if err := applyRemoveTaintOnNode(context.Background(), baseClient, node); err != nil {
		t.Fatalf("did not expect error when node has no taints: %v", err)
	}
}

func TestApplyRemoveTaintOnNodeSuccess(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "success"},
		Spec:       corev1.NodeSpec{Taints: []corev1.Taint{{Key: "k", Value: "v", Effect: corev1.TaintEffectNoSchedule}}},
	}
	remover := &nodesv1alpha1.TaintRemover{
		ObjectMeta: metav1.ObjectMeta{Name: "success-remover"},
		Spec:       nodesv1alpha1.TaintRemoverSpec{Taints: []corev1.Taint{{Key: "k", Value: "v", Effect: corev1.TaintEffectNoSchedule}}},
	}
	client := fake.NewClientBuilder().WithScheme(unitTestScheme).WithObjects(node.DeepCopy(), remover).Build()

	if err := applyRemoveTaintOnNode(context.Background(), client, node); err != nil {
		t.Fatalf("unexpected error removing taints: %v", err)
	}

	updated := &corev1.Node{}
	if err := client.Get(context.Background(), types.NamespacedName{Name: "success"}, updated); err != nil {
		t.Fatalf("failed to fetch node: %v", err)
	}
	for _, tnt := range updated.Spec.Taints {
		if tnt.Key == "k" && tnt.Value == "v" {
			t.Fatalf("expected taint to be removed")
		}
	}
}

func TestPatchNodesToRemoveTaintsSuccess(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "patched"},
		Spec:       corev1.NodeSpec{Taints: []corev1.Taint{{Key: "k", Effect: corev1.TaintEffectNoSchedule}}},
	}
	baseClient := fake.NewClientBuilder().WithScheme(unitTestScheme).WithObjects(node.DeepCopy()).Build()

	removed, err := patchNodesToRemoveTaints(context.Background(), baseClient, []*corev1.Node{node}, []*corev1.Taint{{Key: "k", Effect: corev1.TaintEffectNoSchedule}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected one node to be patched, got %d", removed)
	}

	// Verify taint removed in cluster
	updated := &corev1.Node{}
	if err := baseClient.Get(context.Background(), types.NamespacedName{Name: "patched"}, updated); err != nil {
		t.Fatalf("failed to fetch node: %v", err)
	}
	for _, tnt := range updated.Spec.Taints {
		if tnt.Key == "k" {
			t.Fatalf("expected taint to be removed")
		}
	}
}
