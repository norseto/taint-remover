package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var _ = Describe("NodeHandler", func() {
	var (
		ctx        context.Context
		reconciler *TaintRemoverReconciler
		handler    *nodeHandler
		node       *corev1.Node
		queue      workqueue.RateLimitingInterface
	)

	BeforeEach(func() {
		ctx = context.TODO()
		scheme := runtime.NewScheme()
		reconciler = &TaintRemoverReconciler{
			Client: k8sClient,
			Scheme: scheme,
		}
		handler = &nodeHandler{r: reconciler}
		node = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node-handler",
			},
			Spec: corev1.NodeSpec{
				Taints: []corev1.Taint{
					{
						Key:    "test-key",
						Value:  "test-value",
						Effect: corev1.TaintEffectNoSchedule,
					},
				},
			},
		}
		// Create the node in the cluster
		Expect(k8sClient.Create(ctx, node)).To(Succeed())

		// Create a mock queue
		queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	})

	AfterEach(func() {
		// Delete the node
		Expect(k8sClient.Delete(ctx, node)).To(Succeed())
	})

	Describe("Create", func() {
		It("should call applyRemoveTaintOnNode", func() {
			// Call the Create method
			handler.Create(ctx, event.CreateEvent{Object: node}, queue)

			// Verification is implicit - if there's no panic, the test passes
			// The actual functionality is tested in the applyRemoveTaintOnNode tests
		})
	})

	Describe("Update", func() {
		It("should call applyRemoveTaintOnNode", func() {
			// Call the Update method
			handler.Update(ctx, event.UpdateEvent{ObjectOld: node, ObjectNew: node}, queue)

			// Verification is implicit - if there's no panic, the test passes
			// The actual functionality is tested in the applyRemoveTaintOnNode tests
		})
	})

	Describe("Delete", func() {
		It("should do nothing", func() {
			// Call the Delete method
			handler.Delete(ctx, event.DeleteEvent{Object: node}, queue)

			// No-op function, so just verify it doesn't panic
		})
	})

	Describe("Generic", func() {
		It("should do nothing", func() {
			// Call the Generic method
			handler.Generic(ctx, event.GenericEvent{Object: node}, queue)

			// No-op function, so just verify it doesn't panic
		})
	})
})
