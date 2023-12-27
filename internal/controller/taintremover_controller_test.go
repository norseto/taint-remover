package controller

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/norseto/taint-remover/api/v1alpha1"
	nodesv1alpha1 "github.com/norseto/taint-remover/api/v1alpha1"
)

var _ = Describe("TaintRemoverReconciler", func() {
	var (
		ctx    context.Context
		client client.Client
		scheme *runtime.Scheme
		tr     *v1alpha1.TaintRemover
		node   *corev1.Node
	)

	BeforeEach(func() {
		ctx = context.TODO()
		client = k8sClient
		scheme = runtime.NewScheme()
		tr = nil
		node = nil
	})

	AfterEach(func() {
		if tr != nil {
			Expect(client.Delete(ctx, tr)).To(Succeed())
		}
		if node != nil {
			Expect(client.Delete(ctx, node)).To(Succeed())
		}
	})

	Describe("Reconcile", func() {
		Context("When there are taints and nodes", func() {
			It("should remove taints from nodes", func() {
				node, tr = setupNodeAndRemover(fooBarTaint, fooBarTaint)

				// Reconcile the TaintRemover object
				reconciler := &TaintRemoverReconciler{
					Client: client,
					Scheme: scheme,
				}
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: tr.Name,
					},
				}
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Verify that the taints have been removed from the node
				nodeKey := types.NamespacedName{
					Name: node.Name,
				}
				Expect(client.Get(ctx, nodeKey, node)).To(Succeed())
				Expect(node.Spec.Taints).To(HaveLen(1))
			})
		})

		Context("When there are no taints in remover", func() {
			It("should not remove taints from nodes", func() {
				node, tr = setupNodeAndRemover(fooBarTaint, emptyTait)

				// Reconcile the TaintRemover object
				reconciler := &TaintRemoverReconciler{
					Client: client,
					Scheme: scheme,
				}
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: tr.Name,
					},
				}
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())

				// Verify that the taints have not been removed from the node
				nodeKey := types.NamespacedName{
					Name: node.Name,
				}
				Expect(client.Get(ctx, nodeKey, node)).To(Succeed())
				Expect(node.Spec.Taints).To(HaveLen(2))
			})
		})

		Context("When there are no nodes", func() {
			It("should not remove taints", func() {
				// Create a TaintRemover object
				tr = createTaintRemover(fooBarTaint)

				// Reconcile the TaintRemover object
				reconciler := &TaintRemoverReconciler{
					Client: client,
					Scheme: scheme,
				}
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: tr.Name,
					},
				}
				_, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

var _ = Describe("SetupWithManager", func() {
	var (
		ctx    context.Context
		client client.Client
		scheme *runtime.Scheme
		mgr    ctrl.Manager
	)
	BeforeEach(func() {
		// Create a new manager
		var err error

		mgr, err = ctrl.NewManager(cfg, ctrl.Options{})
		Expect(err).NotTo(HaveOccurred())

		// Create a new reconciler
		r := &TaintRemoverReconciler{
			Client: client,
			Scheme: scheme,
		}

		// Setup the reconciler with the manager
		err = r.SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should set up watches", func() {
		// Check if the reconciler is watching TaintRemover objects
		watches, err := mgr.GetCache().GetInformer(ctx, &nodesv1alpha1.TaintRemover{})
		Expect(err).NotTo(HaveOccurred())
		Expect(watches).To(Not(BeNil()))

		// Check if the reconciler is watching Node objects
		watches, err = mgr.GetCache().GetInformer(ctx, &corev1.Node{})
		Expect(err).NotTo(HaveOccurred())
		Expect(watches).To(Not(BeNil()))
	})
})

var _ = Describe("internalMethods", func() {
	var (
		ctx    context.Context
		client client.Client
		scheme *runtime.Scheme
		tr     *v1alpha1.TaintRemover
		node   *corev1.Node
	)

	BeforeEach(func() {
		ctx = context.TODO()
		client = k8sClient
		scheme = runtime.NewScheme()
		tr = nil
		node = nil
	})

	AfterEach(func() {
		if tr != nil {
			Expect(client.Delete(ctx, tr)).To(Succeed())
		}
		if node != nil {
			Expect(client.Delete(ctx, node)).To(Succeed())
		}
	})

	Describe("applyTaintRemoveOnNode", func() {
		Context("When there are taints and nodes", func() {
			It("should remove taints from nodes", func() {
				// Create a TaintRemover object
				node, tr = setupNodeAndRemover(fooBarTaint, fooBarTaint)

				// Reconcile the TaintRemover object
				reconciler := &TaintRemoverReconciler{
					Client: client,
					Scheme: scheme,
				}
				err := reconciler.applyTaintRemoveOnNode(ctx, node)
				Expect(err).NotTo(HaveOccurred())

				// Verify that the taints have been removed from the node
				nodeKey := types.NamespacedName{
					Name: node.Name,
				}
				Expect(client.Get(ctx, nodeKey, node)).To(Succeed())
				Expect(node.Spec.Taints).To(HaveLen(1))
			})
		})

		Context("When there are no taints in remover", func() {
			It("should not remove taints from nodes", func() {
				// Create a TaintRemover object
				node, tr = setupNodeAndRemover(fooBarTaint, emptyTait)

				// Reconcile the TaintRemover object
				reconciler := &TaintRemoverReconciler{
					Client: client,
					Scheme: scheme,
				}
				err := reconciler.applyTaintRemoveOnNode(ctx, node)
				Expect(err).NotTo(HaveOccurred())

				// Verify that the taints have not been removed from the node
				nodeKey := types.NamespacedName{
					Name: node.Name,
				}
				Expect(client.Get(ctx, nodeKey, node)).To(Succeed())
				Expect(node.Spec.Taints).To(HaveLen(2))
			})
		})

		Context("When there are no remover", func() {
			It("should not remove taints from nodes", func() {
				// Create a TaintRemover object
				node = createNodeWithTaints(fooBarTaint)

				// Reconcile the TaintRemover object
				reconciler := &TaintRemoverReconciler{
					Client: client,
					Scheme: scheme,
				}
				err := reconciler.applyTaintRemoveOnNode(ctx, node)
				Expect(err).NotTo(HaveOccurred())

				// Verify that the taints have not been removed from the node
				nodeKey := types.NamespacedName{
					Name: node.Name,
				}
				Expect(client.Get(ctx, nodeKey, node)).To(Succeed())
				Expect(node.Spec.Taints).To(HaveLen(2))
			})
		})
	})
})

var fooBarTaint = []corev1.Taint{
	{
		Key:    "foo",
		Value:  "bar",
		Effect: "NoSchedule",
	},
}
var emptyTait []corev1.Taint

// setupNodeAndRemover creates a new Node object and a new TaintRemover object and returns them.
// It uses the createTaintRemover and createNodeWithTaints functions to create these objects.
// The taints provided as input are used to create the TaintRemover object and the node is created
// with taints specified by node parameter.
// The newly created TaintRemover and Node objects are then returned.
func setupNodeAndRemover(node, remover []corev1.Taint) (*corev1.Node, *nodesv1alpha1.TaintRemover) {
	tr := createTaintRemover(remover)
	n := createNodeWithTaints(node)

	return n, tr
}

// createTaintRemover creates a new TaintRemover object and adds it to the cluster.
func createTaintRemover(taints []corev1.Taint) *v1alpha1.TaintRemover {
	ctx := context.TODO()
	tr := &v1alpha1.TaintRemover{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-taint-remover",
		},
		Spec: v1alpha1.TaintRemoverSpec{
			Taints: taints,
		},
	}
	Expect(k8sClient.Create(ctx, tr)).To(Succeed())
	return tr
}

// createNodeWithTaints creates a new Node object with specified taints
// and adds it to the cluster.
func createNodeWithTaints(taints []corev1.Taint) *corev1.Node {
	ctx := context.TODO()
	n := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Spec: corev1.NodeSpec{
			Taints: taints,
		},
	}
	Expect(k8sClient.Create(ctx, n)).To(Succeed())
	// not-ready taint will be added.
	Expect(n.Spec.Taints).To(HaveLen(2))
	return n
}
