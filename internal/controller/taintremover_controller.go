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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nodesv1alpha1 "github.com/norseto/taint-remover/api/v1alpha1"
)

// TaintRemoverReconciler reconciles a TaintRemover object
type TaintRemoverReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=nodes.peppy-ratio.dev,resources=taintremovers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nodes.peppy-ratio.dev,resources=taintremovers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nodes.peppy-ratio.dev,resources=taintremovers/finalizers,verbs=update

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
	log.Info("Got taints", "taints", taints)

	nodes, err := r.getTargetNodes(ctx, req.NamespacedName)
	if err != nil {
		log.Error(err, "Failed to get nodes")
	}
	if len(nodes) < 1 {
		return reconcile.Result{}, nil
	}
	log.Info("Got nodes", "tainted nodes", len(nodes))
	err = r.removeTaints(ctx, nodes, taints)
	if err != nil {
		log.Error(err, "Failed to remove taints")
	}

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaintRemoverReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(r)
	if err != nil {
		return nil
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&nodesv1alpha1.TaintRemover{}).
		Complete(r)
}

func (r *TaintRemoverReconciler) getTaints(ctx context.Context) ([]corev1.Taint, error) {
	log := log.FromContext(ctx)

	removers := &nodesv1alpha1.TaintRemoverList{}
	err := r.Client.List(ctx, removers)
	if err != nil {
		log.Error(err, "Failed to get Remover")
		return nil, err
	}
	if len(removers.Items) < 1 {
		return nil, nil
	}

	var taints []corev1.Taint
	for _, v := range removers.Items {
		taints = append(taints, v.Spec.Taints...)
	}

	return taints, nil
}

func (r *TaintRemoverReconciler) getTargetNodes(ctx context.Context, name types.NamespacedName) ([]corev1.Node, error) {
	var nodes []corev1.Node

	// Check Reconcile target is a node
	node := corev1.Node{}
	err := r.Get(ctx, name, &node)
	if err == nil && len(node.Spec.Taints) > 0 {
		return append(nodes, node), nil
	}
	if !errors.IsNotFound(err) {
		return nil, err
	}
	list := &corev1.NodeList{}
	err = r.Client.List(ctx, list)
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

func (r *TaintRemoverReconciler) removeTaints(ctx context.Context, nodes []corev1.Node, taints []corev1.Taint) error {
	log := log.FromContext(ctx)

	for _, n := range nodes {
		node := &n
		needPatch := false
		for _, t := range taints {
			newNode, changed, err := tutil.RemoveTaint(node, &t)
			if err != nil {
				return err
			}
			node = newNode
			needPatch = (needPatch || changed)
		}
		log.Info("Taint check", "NeedPatch", needPatch)
		if needPatch {
			data, err := json.Marshal(node.Spec.Taints)
			if err != nil {
				return err
			}
			patch := client.RawPatch(types.MergePatchType, data)
			r.Client.Patch(ctx, &n, patch)
		}
	}
	return nil
}
