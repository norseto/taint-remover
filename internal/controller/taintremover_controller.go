/*
MIT License

Copyright (c) 2023 Norihiro Seto

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

	tutil "github.com/norseto/taint-remover/internal/taints"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	nodesv1alpha1 "github.com/norseto/taint-remover/api/v1alpha1"
)

// TaintRemoverReconciler reconciles a TaintRemover object
type TaintRemoverReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type patchNodeSpec struct {
	Taints []corev1.Taint `json:"taints"`
}
type patchNode struct {
	Spec patchNodeSpec `json:"spec"`
}

//+kubebuilder:rbac:groups=nodes.peppy-ratio.dev,resources=taintremovers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nodes.peppy-ratio.dev,resources=taintremovers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nodes.peppy-ratio.dev,resources=taintremovers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the TaintRemover object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.0/pkg/reconcile
func (r *TaintRemoverReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	taints, err := r.getTaints(ctx)
	if err != nil {
		log.Error(err, "Failed to get config")
	}
	if len(taints) < 1 {
		return reconcile.Result{}, nil
	}
	log.Info("Got CRD targets", "taints", taints)

	nodes, err := r.getTargetNodes(ctx)
	if err != nil {
		log.Error(err, "Failed to get nodes")
	}
	if len(nodes) < 1 {
		return reconcile.Result{}, nil
	}
	log.Info("Got nodes", "tainted nodes", len(nodes))
	removed, err := r.removeTaints(ctx, nodes, taints)
	if err != nil {
		log.Error(err, "Failed to remove taints")
	}
	log.Info("removed taints", "removed", removed)

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaintRemoverReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nodesv1alpha1.TaintRemover{}).
		WatchesRawSource(source.Kind(mgr.GetCache(), &corev1.Node{}),
			&nodeHandler{r: r},
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Complete(r)
}

// findTaintsForNode finds target taints and remove
func (r *TaintRemoverReconciler) findTaintsForNode(ctx context.Context, node client.Object) error {
	var nodes []corev1.Node

	log := log.FromContext(ctx)

	log.Info("findTaintsForNode starting", "node", node.GetName(), "resver", node.GetResourceVersion())
	found := &corev1.Node{}
	criterion := types.NamespacedName{
		Name: node.GetName(),
	}

	err := r.Get(ctx, criterion, found)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "could not find the node", "criteria", criterion)
		}
		return err
	}

	if len(found.Spec.Taints) > 0 {
		nodes = append(nodes, *found.DeepCopy())
	}

	taints, err := r.getTaints(ctx)
	if err != nil {
		log.Error(err, "failed to get taints")
		return err
	}

	log.Info("findTaintsForNode", "node taints", len(found.Spec.Taints), "target taints", len(taints))
	removed, err := r.removeTaints(ctx, nodes, taints)
	if err != nil {
		log.Error(err, "failed to remove taints")
		return err
	}
	log.Info("removed taints", "removed", removed)

	return nil
}

// getTaints gets all remove target taints in all TaintRemovers
func (r *TaintRemoverReconciler) getTaints(ctx context.Context) ([]corev1.Taint, error) {
	log := log.FromContext(ctx)

	removers := &nodesv1alpha1.TaintRemoverList{}
	err := r.List(ctx, removers)
	if err != nil {
		log.Error(err, "Failed to get Remover")
		return nil, err
	}
	if len(removers.Items) < 1 {
		return nil, nil
	}

	var taints []corev1.Taint
	for _, v := range removers.Items {
		for _, t := range v.Spec.Taints {
			if tutil.TaintExists(taints, &t) {
				continue
			}
			taints = append(taints, t)
		}
	}

	return taints, nil
}

// getTargetNodes gets all nodes that has any taints.
func (r *TaintRemoverReconciler) getTargetNodes(ctx context.Context) ([]corev1.Node, error) {
	var nodes []corev1.Node

	list := &corev1.NodeList{}
	err := r.List(ctx, list)
	if err != nil {
		return nil, err
	}

	for _, v := range list.Items {
		if len(v.Spec.Taints) > 0 {
			nodes = append(nodes, *v.DeepCopy())
		}
	}

	return nodes, err
}

// removeTaints removes all taints from target nodes
func (r *TaintRemoverReconciler) removeTaints(ctx context.Context, nodes []corev1.Node, taints []corev1.Taint) (int, error) {
	log := log.FromContext(ctx)
	removed := 0

	for _, n := range nodes {
		node := &n
		needPatch := false
		for _, t := range taints {
			newNode, changed, err := tutil.RemoveTaint(node, &t)
			if err != nil {
				return removed, err
			}
			node = newNode
			needPatch = (needPatch || changed)
		}
		log.Info("Taint check", "NeedPatch", needPatch)
		if needPatch {
			patchNode := patchNode{Spec: patchNodeSpec{Taints: node.Spec.Taints}}
			data, err := json.Marshal(patchNode)
			if err != nil {
				return removed, err
			}
			log.Info("Taint remove", "Patch", string(data))
			patch := client.RawPatch(types.StrategicMergePatchType, data)
			r.Patch(ctx, &n, patch)
			removed++
		}
	}
	return removed, nil
}

type nodeHandler struct {
	r *TaintRemoverReconciler
}

func (nh *nodeHandler) Create(ctx context.Context, evt event.CreateEvent, rlmit workqueue.RateLimitingInterface) {
	nh.r.findTaintsForNode(ctx, evt.Object)
}
func (nh *nodeHandler) Update(ctx context.Context, evt event.UpdateEvent, rlmit workqueue.RateLimitingInterface) {
	nh.r.findTaintsForNode(ctx, evt.ObjectNew)
}
func (nh *nodeHandler) Delete(ctx context.Context, evt event.DeleteEvent, rlmit workqueue.RateLimitingInterface) {
	// No-op
}
func (nh *nodeHandler) Generic(ctx context.Context, evt event.GenericEvent, rlmit workqueue.RateLimitingInterface) {
	// No-op
}
