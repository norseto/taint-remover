/*
MIT License

Copyright (c) 2023-2025 Norihiro Seto

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controller

import (
	"context"
	"encoding/json"
	"fmt"

	tutil "github.com/norseto/taint-remover/internal/taints"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nodesv1alpha1 "github.com/norseto/taint-remover/api/v1alpha1"
)

// TaintRemoverReconciler reconciles a TaintRemover object
type TaintRemoverReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// nodeTaintsPatchSpec represents a node object and its patch.
type nodeTaintsPatchSpec struct {
	node  *corev1.Node
	patch *nodeSpecPatchPayload
}

// nodeSpecPatchSpec defines the specification for patching a node's taints.
type nodeSpecPatchSpec struct {
	Taints []corev1.Taint `json:"taints"`
}

// nodeSpecPatchPayload represents a patch for a node object
type nodeSpecPatchPayload struct {
	Spec nodeSpecPatchSpec `json:"spec"`
}

//+kubebuilder:rbac:groups=nodes.peppy-ratio.dev,resources=taintremovers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nodes.peppy-ratio.dev,resources=taintremovers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nodes.peppy-ratio.dev,resources=taintremovers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *TaintRemoverReconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	taints, err := getAllTaintsToRemove(ctx, r.Client)
	if err != nil {
		logger.Error(err, "Failed to get config")
		return ctrl.Result{}, err
	}
	if len(taints) < 1 {
		return reconcile.Result{}, nil
	}
	logger.V(1).Info("Got CRD targets", "taints", taints)

	nodes, err := getTaintedNodes(ctx, r.Client)
	if err != nil {
		logger.Error(err, "Failed to get nodes")
		return ctrl.Result{}, err
	}
	if len(nodes) < 1 {
		return reconcile.Result{}, nil
	}
	logger.V(1).Info("Got nodes", "tainted nodes", len(nodes))
	removed, err := patchNodesToRemoveTaints(ctx, r.Client, nodes, taints)
	if err != nil {
		logger.Error(err, "Failed to remove taints")
	}
	logger.V(1).Info("removed taints", "removed", removed)

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaintRemoverReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nodesv1alpha1.TaintRemover{}).
		Watches(&corev1.Node{}, &nodeHandler{r: r},
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Complete(r)
}

// applyRemoveTaintOnNode applies the removed taints on the new or updated Node.
func applyRemoveTaintOnNode(ctx context.Context, c client.Client, node client.Object) error {
	logger := log.FromContext(ctx)
	logger.Info("applyRemoveTaintOnNode starting", "node", node.GetName(), "resver", node.GetResourceVersion())

	found, err := getNodeAndCheckTaints(ctx, c, node)
	if err != nil || found == nil {
		logger.V(2).Info("node not found or no taints", "node", node.GetName())
		return err
	}

	nodes := []*corev1.Node{found.DeepCopy()}
	taints, err := getAllTaintsToRemove(ctx, c)
	if err != nil {
		logger.Error(err, "failed to get taints")
		return err
	}
	logger.Info("applyRemoveTaintOnNode", "node taints", len(found.Spec.Taints), "target taints", len(taints))

	removed, err := patchNodesToRemoveTaints(ctx, c, nodes, taints)
	if err != nil {
		logger.Error(err, "failed to remove taints")
		return err
	}
	logger.Info("removed taints", "removed", removed)
	return nil
}

// getNodeAndCheckTaints retrieves the specified node object and checks if it has any taints.
// If the node is not found or does not have any taints, it returns nil.
// Otherwise, it returns the node object.
func getNodeAndCheckTaints(ctx context.Context, c client.Client, node client.Object) (*corev1.Node, error) {
	logger := log.FromContext(ctx)
	criterion := types.NamespacedName{
		Name: node.GetName(),
	}
	found := &corev1.Node{}

	err := c.Get(ctx, criterion, found)
	if err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "could not find the node", "criteria", criterion)
		}
		return nil, err
	}

	if len(found.Spec.Taints) < 1 {
		return nil, nil
	}

	return found, nil
}

// getAllTaintsToRemove retrieves the list of taints from the TaintRemover objects in the cluster.
func getAllTaintsToRemove(ctx context.Context, c client.Client) ([]*corev1.Taint, error) {
	logger := log.FromContext(ctx)

	removers := &nodesv1alpha1.TaintRemoverList{}
	err := c.List(ctx, removers)
	if err != nil {
		logger.Error(err, "Failed to get Remover")
		return nil, err
	}
	if len(removers.Items) < 1 {
		return nil, nil
	}

	// Use a set to efficiently track unique taints (key:effect)
	uniqueTaintKeys := sets.New[string]()
	var taints []corev1.Taint

	for _, v := range removers.Items {
		for _, t := range v.Spec.Taints {
			// Create a unique key for the taint based on Key and Effect
			taintKey := fmt.Sprintf("%s:%s", t.Key, t.Effect)
			if uniqueTaintKeys.Has(taintKey) {
				continue // Skip if already added
			}
			uniqueTaintKeys.Insert(taintKey)
			taints = append(taints, t)
		}
	}

	return ConvertToPointerArray(taints), nil
}

// ConvertToPointerArray converts a slice of type T to a slice of pointers to T
func ConvertToPointerArray[T any](arr []T) []*T {
	result := make([]*T, len(arr))
	for i := range arr {
		result[i] = &arr[i]
	}
	return result
}

// getTaintedNodes retrieves a list of nodes that have taints applied.
// It queries the cluster for all nodes and checks if each node has any taints.
// If a node has taints, it adds a deep copy of the node to the list of target nodes.
//
// This function returns the list of target nodes and an error, if any.
// If the cluster query fails, it returns a nil slice of nodes and the error.
func getTaintedNodes(ctx context.Context, c client.Client) ([]*corev1.Node, error) {
	var nodes []*corev1.Node

	list := &corev1.NodeList{}
	err := c.List(ctx, list)
	if err != nil {
		return nil, err
	}

	for _, v := range list.Items {
		if len(v.Spec.Taints) > 0 {
			nodes = append(nodes, v.DeepCopy())
		}
	}

	return nodes, err
}

// patchNodesToRemoveTaints applies patches to target nodes to remove the specified taints.
// It processes all nodes even if errors occur for some, returning the last encountered error.
func patchNodesToRemoveTaints(ctx context.Context, c client.Client, nodes []*corev1.Node, taints []*corev1.Taint) (int, error) {
	logger := log.FromContext(ctx)
	removed := 0
	var lastErr error // Keep track of the last error encountered

	patches := makePatches(nodes, taints)
	for _, n := range patches {
		logger.Info("Removing taints from node", "node", n.node.Name)
		if err := patchNode(ctx, c, n.node, n.patch); err != nil {
			logger.Error(err, "Failed to patch node", "node", n.node.Name)
			lastErr = err // Store the error and continue
			// Continue processing other nodes, but ensure we return the error
			// at the end so the reconcile loop retries.
			continue
		}
		removed++
	}
	// Return the number of nodes successfully patched and the last error encountered (if any)
	return removed, lastErr
}

// makePatches calculates the patches required to remove the specified taints from each node.
func makePatches(nodes []*corev1.Node, taints []*corev1.Taint) []nodeTaintsPatchSpec {
	result := make([]nodeTaintsPatchSpec, 0, len(nodes))

	for _, n := range nodes {
		newTaints, needPatch := makeNewTaintsForNode(n, taints)
		if !needPatch {
			continue
		}
		patch := nodeSpecPatchPayload{Spec: nodeSpecPatchSpec{Taints: newTaints}}
		result = append(result, nodeTaintsPatchSpec{node: n.DeepCopy(), patch: &patch})
	}
	return result
}

// makeNewTaintsForNode removes the specified taints from the target node.
// It returns the updated list of taints after removing the specified taints,
// as well as a boolean indicating whether any taints were removed.
func makeNewTaintsForNode(target *corev1.Node, taints []*corev1.Taint) ([]corev1.Taint, bool) {
	if target == nil {
		return nil, false
	}
	nodeTaints := target.Spec.Taints
	deleted := false
	for _, taint := range taints {
		if !tutil.TaintExists(nodeTaints, taint) {
			continue
		}
		var taintDeleted bool
		nodeTaints, taintDeleted = tutil.DeleteTaint(nodeTaints, taint)
		deleted = deleted || taintDeleted
	}
	return nodeTaints, deleted
}

// patchNode patches the specified node object with the given patch.
func patchNode(ctx context.Context, c client.Client, node *corev1.Node, patch *nodeSpecPatchPayload) error {
	logger := log.FromContext(ctx)

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal node patch for node %s: %w", node.Name, err)
	}
	logger.Info("Apply node patch", "Patch", string(patchBytes))
	raw := client.RawPatch(types.StrategicMergePatchType, patchBytes)
	return c.Patch(ctx, node, raw)
}

// nodeHandler implements event.EventHandler for Node objects.
// It triggers reconciliation of potentially affected TaintRemover objects
// when a Node is created or updated.
type nodeHandler struct {
	r *TaintRemoverReconciler
}

func (nh *nodeHandler) Create(ctx context.Context, evt event.TypedCreateEvent[client.Object], _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if err := applyRemoveTaintOnNode(ctx, nh.r.Client, evt.Object); err != nil {
		log.FromContext(ctx).Error(err, "Failed to apply taint removal on node creation", "node", evt.Object.GetName())
	}
}

func (nh *nodeHandler) Update(ctx context.Context, evt event.TypedUpdateEvent[client.Object], _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if err := applyRemoveTaintOnNode(ctx, nh.r.Client, evt.ObjectNew); err != nil {
		log.FromContext(ctx).Error(err, "Failed to apply taint removal on node update", "node", evt.ObjectNew.GetName())
	}
}

func (nh *nodeHandler) Delete(context.Context, event.TypedDeleteEvent[client.Object], workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// No-op
}

func (nh *nodeHandler) Generic(context.Context, event.TypedGenericEvent[client.Object], workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// No-op
}
