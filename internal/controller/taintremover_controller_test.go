package controller

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/norseto/taint-remover/api/v1alpha1"
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
				// Create a TaintRemover object
				tr = &v1alpha1.TaintRemover{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-taint-remover",
					},
					Spec: v1alpha1.TaintRemoverSpec{
						Taints: []corev1.Taint{
							{
								Key:    "foo",
								Value:  "bar",
								Effect: "NoSchedule",
							},
						},
					},
				}
				Expect(client.Create(ctx, tr)).To(Succeed())

				// Create a Node object with taints
				node = &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
					Spec: corev1.NodeSpec{
						Taints: []corev1.Taint{
							{
								Key:    "foo",
								Value:  "bar",
								Effect: "NoSchedule",
							},
						},
					},
				}
				Expect(client.Create(ctx, node)).To(Succeed())
				// not-ready taint will be added.
				Expect(node.Spec.Taints).To(HaveLen(2))

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

		Context("When there are no taints", func() {
			It("should not remove taints from nodes", func() {
				// Create a TaintRemover object
				tr = &v1alpha1.TaintRemover{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-taint-remover",
					},
					Spec: v1alpha1.TaintRemoverSpec{
						Taints: []corev1.Taint{},
					},
				}
				Expect(client.Create(ctx, tr)).To(Succeed())

				// Create a Node object with taints
				node = &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-node",
					},
					Spec: corev1.NodeSpec{
						Taints: []corev1.Taint{
							{
								Key:    "foo",
								Value:  "bar",
								Effect: "NoSchedule",
							},
						},
					},
				}
				Expect(client.Create(ctx, node)).To(Succeed())
				Expect(node.Spec.Taints).To(HaveLen(2))

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
				tr = &v1alpha1.TaintRemover{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-taint-remover",
					},
					Spec: v1alpha1.TaintRemoverSpec{
						Taints: []corev1.Taint{
							{
								Key:    "foo",
								Value:  "bar",
								Effect: "NoSchedule",
							},
						},
					},
				}
				Expect(client.Create(ctx, tr)).To(Succeed())

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
