/*
MIT License

Copyright (c) 2025 ngrok, Inc.

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
package service

import (
	"fmt"
	"math/rand/v2"
	"time"

	ngrokv1alpha1 "github.com/ngrok/ngrok-operator/api/ngrok/v1alpha1"
	"github.com/ngrok/ngrok-operator/internal/annotations"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LoadBalancer = corev1.ServiceTypeLoadBalancer
	ClusterIP    = corev1.ServiceTypeClusterIP
)

// getCloudEndpoints fetches CloudEndpoints in the given namespace
func getCloudEndpoints(k8sClient client.Client, namespace string) (*ngrokv1alpha1.CloudEndpointList, error) {
	clepList := &ngrokv1alpha1.CloudEndpointList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
	}
	err := k8sClient.List(ctx, clepList, listOpts...)
	return clepList, err
}

// getAgentEndpoints fetches AgentEndpoints in the given namespace
func getAgentEndpoints(k8sClient client.Client, namespace string) (*ngrokv1alpha1.AgentEndpointList, error) {
	aepList := &ngrokv1alpha1.AgentEndpointList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
	}
	err := k8sClient.List(ctx, aepList, listOpts...)
	return aepList, err
}

var _ = Describe("ServiceController", func() {
	const (
		timeout  = 10 * time.Second
		duration = 10 * time.Second
		interval = 250 * time.Millisecond
	)

	var (
		namespace string = "default"
		svc       *corev1.Service
		svcName   string
		svcType   corev1.ServiceType
	)

	BeforeEach(func() {
		svcName = fmt.Sprintf("test-service-%d", rand.IntN(100000))
		svc = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svcName,
				Namespace: namespace,
			},
			Spec: corev1.ServiceSpec{
				Type: svcType,
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

	JustBeforeEach(func() {
		Expect(k8sClient.Create(ctx, svc)).To(Succeed())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, svc)).To(Succeed())
	})

	When("the service type is not a LoadBalancer", func() {
		BeforeEach(func() { svcType = ClusterIP })

		It("should ignore the service", func() {
			Consistently(func(g Gomega) {
				fetched := &corev1.Service{}
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(svc), fetched)
				g.Expect(err).NotTo(HaveOccurred())

				By("By checking the service is not modified")
				g.Expect(fetched.Finalizers).To(BeEmpty())
			}, duration, interval).Should(Succeed())
		})
	})

	When("service type is LoadBalancer", func() {
		BeforeEach(func() { svcType = LoadBalancer })

		When("the service has a non-ngrok load balancer class", func() {
			It("should ignore the service", func() {
				Consistently(func(g Gomega) {
					fetched := &corev1.Service{}
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(svc), fetched)
					g.Expect(err).NotTo(HaveOccurred())

					By("By checking the service is not modified")
					g.Expect(fetched.Finalizers).To(BeEmpty())
				}, duration, interval).Should(Succeed())
			})
		})

		When("the service has the ngrok load balancer class", func() {
			BeforeEach(func() { svc.Spec.LoadBalancerClass = ptr.To("ngrok") })

			It("should have a finalizer added", func() {
				Eventually(func(g Gomega) {
					fetched := &corev1.Service{}
					err := k8sClient.Get(ctx, client.ObjectKeyFromObject(svc), fetched)
					g.Expect(err).NotTo(HaveOccurred())

					By("By checking the service has a finalizer added")
					g.Expect(fetched.Finalizers).To(ContainElement("k8s.ngrok.com/finalizer"))
				}, timeout, interval).Should(Succeed())
			})

			When("the service does not have a URL annotation", func() {
				It("Should reserve a TCP address", func() {
					Eventually(func(g Gomega) {
						fetched := &corev1.Service{}
						err := k8sClient.Get(ctx, client.ObjectKeyFromObject(svc), fetched)
						g.Expect(err).NotTo(HaveOccurred())

						By("By checking the service has a URL annotation")
						GinkgoLogr.Info("Got service", "fetched", fetched)
						urlAnnotation, exists := fetched.Annotations["ngrok.com/url"]
						g.Expect(exists).To(BeTrue())
						g.Expect(urlAnnotation).To(MatchRegexp(`^tcp://[a-zA-Z0-9\-\.]+:\d+$`))
					}, timeout, interval).Should(Succeed())
				})
			})

			When("endpoints verbose", func() {
				BeforeEach(func() {
					svc.Annotations[annotations.MappingStrategyAnnotation] = string(annotations.MappingStrategy_EndpointsVerbose)
				})

				It("Should create a cloud endpoint", func() {
					Eventually(func(g Gomega) {
						cleps, err := getCloudEndpoints(k8sClient, namespace)
						g.Expect(err).NotTo(HaveOccurred())

						By("By checking a cloud endpoint exists")
						g.Expect(cleps.Items).To(HaveLen(1))
					}, timeout, interval).Should(Succeed())
				})
			})
		})
	})
})
