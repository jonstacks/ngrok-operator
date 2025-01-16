package ingress

import (
	"context"
	"time"

	ingressv1alpha1 "github.com/ngrok/ngrok-operator/api/ingress/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newTestService(isLoadBalancer bool, isOurLoadBalancerClass bool, annotations map[string]string) *corev1.Service {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-service",
			Namespace:   "test-namespace",
			Annotations: annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:     "tcp",
					Protocol: corev1.ProtocolTCP,
					Port:     80,
				},
			},
		},
	}
	if isLoadBalancer {
		svc.Spec.Type = corev1.ServiceTypeLoadBalancer
	}
	if isOurLoadBalancerClass {
		svc.Spec.LoadBalancerClass = ptr.To(NgrokLoadBalancerClass)
	} else {
		svc.Spec.LoadBalancerClass = ptr.To("not-ngrok")
	}

	return svc
}

var _ = Describe("ServiceController", func() {
	const (
		timeout  = 10 * time.Second
		duration = 10 * time.Second
		interval = 250 * time.Millisecond
	)
	var (
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
			},
		}
		testService *corev1.Service
	)

	BeforeEach(func() {
		ctx := context.Background()
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

		testService = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-service",
				Namespace: namespace.GetName(),
			},
			Spec: corev1.ServiceSpec{
				Type:              corev1.ServiceTypeLoadBalancer,
				LoadBalancerClass: ptr.To(NgrokLoadBalancerClass),
				Ports: []corev1.ServicePort{
					{
						Name:     "tcp",
						Protocol: corev1.ProtocolTCP,
						Port:     80,
					},
				},
			},
		}
	})

	AfterEach(func() {
		ctx := context.Background()
		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
	})

	DescribeTable("shouldHandleService", func(svc *corev1.Service, expected bool) {
		Expect(shouldHandleService(svc)).To(Equal(expected))
	},
		Entry("Non-LoadBalancer service", newTestService(false, false, nil), false),
		Entry("LoadBalancer service, but not our class", newTestService(true, false, nil), false),
		Entry("LoadBalancer service, our class, but no annotations", newTestService(true, true, nil), true),
	)

	Context("When a ngrok LoadBalancer Service is created", func() {
		It("should create the required ngrok resources", func() {
			By("Creating a Service")
			Expect(k8sClient.Create(ctx, testService)).To(Succeed())

			By("Creating a TCPEdge")
			Eventually(func(g Gomega) {
				edges := &ingressv1alpha1.TCPEdgeList{}
				k8sClient.List(ctx, edges, client.InNamespace(namespace.GetName()))
				g.Expect(edges.Items).To(HaveLen(1))
			})
		})
	})
})
