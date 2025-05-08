package gateway

import (
	"time"

	testutils "github.com/ngrok/ngrok-operator/internal/testutils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func newGateway(gwc *gatewayv1.GatewayClass) *gatewayv1.Gateway {
	return &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testutils.RandomName("gateway"),
			Namespace: "default",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(gwc.Name),
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Port:     80,
					Protocol: gatewayv1.HTTPProtocolType,
				},
			},
		},
	}
}

var _ = Describe("Gateway controller", Ordered, func() {
	const (
		ManagedControllerName   = ControllerName
		UnmanagedControllerName = "k8s.io/some-other-controller"

		timeout  = 10 * time.Second
		duration = 10 * time.Second
		interval = 250 * time.Millisecond
	)

	var (
		gatewayClass *gatewayv1.GatewayClass
		gw           *gatewayv1.Gateway
	)

	When("the gateway's gateway class should be handled by us", func() {
		BeforeAll(func(ctx SpecContext) {
			gatewayClass = testutils.NewGatewayClass(true)
			CreateGatewayClassAndWaitForAcceptance(ctx, gatewayClass, timeout, interval)
		})

		AfterAll(func(ctx SpecContext) {
			DeleteAllGatewayClasses(ctx, timeout, interval)
		})

		BeforeEach(func(ctx SpecContext) {
			gw = newGateway(gatewayClass)
			Expect(k8sClient.Create(ctx, gw)).To(Succeed())
		})

		AfterEach(func(ctx SpecContext) {
			Expect(k8sClient.Delete(ctx, gw)).To(Succeed())
		})

		It("Should accept the gatewway", func(ctx SpecContext) {
			ExpectGatewayAccepted(ctx, gw, timeout, interval)
		})
	})

	When("The gateway's gateway class should not be handled by us", func() {
		BeforeAll(func(ctx SpecContext) {
			gatewayClass = testutils.NewGatewayClass(false)
			Expect(k8sClient.Create(ctx, gatewayClass)).To(Succeed())
		})

		AfterAll(func(ctx SpecContext) {
			DeleteAllGatewayClasses(ctx, timeout, interval)
		})

		BeforeEach(func(ctx SpecContext) {
			gw = newGateway(gatewayClass)
			Expect(k8sClient.Create(ctx, gw)).To(Succeed())
		})

		AfterEach(func(ctx SpecContext) {
			Expect(k8sClient.Delete(ctx, gw)).To(Succeed())
		})

		It("should not accept the gateway", func(ctx SpecContext) {
			Consistently(func(g Gomega) {
				obj := &gatewayv1.Gateway{}
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(gw), obj)).To(Succeed())
				By("Consistently not having an accepted condition with Status True")
				cond := meta.FindStatusCondition(obj.Status.Conditions, string(gatewayv1.GatewayConditionAccepted))
				g.Expect(cond.Status).NotTo(Equal(metav1.ConditionTrue))
			}, timeout, interval).Should(Succeed())
		})
	})
})
