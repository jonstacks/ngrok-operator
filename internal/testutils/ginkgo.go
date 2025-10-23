package testutils

import (
	"context"
	"time"

	ngrokv1alpha1 "github.com/ngrok/ngrok-operator/api/ngrok/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DefaultTimeout  = 20 * time.Second
	DefaultInterval = 500 * time.Millisecond
)

// KGinkgo is a helper for Ginkgo tests that interact with Kubernetes resources.
// It provides methods to assert conditions on Kubernetes objects using Gomega matchers, especially in conjunction with Eventually.
type KGinkgo struct {
	client client.Client
}

// NewKGinkgo creates a new KGinkgo instance
func NewKGinkgo(c client.Client) *KGinkgo {
	return &KGinkgo{
		client: c,
	}
}

type expectOptions struct {
	timeout  time.Duration
	interval time.Duration
}

type KGinkgoOpt func(*expectOptions)

func WithTimeout(timeout time.Duration) KGinkgoOpt {
	return func(o *expectOptions) {
		o.timeout = timeout
	}
}

func WithInterval(interval time.Duration) KGinkgoOpt {
	return func(o *expectOptions) {
		o.interval = interval
	}
}

// ExpectFinalizerToBeAdded asserts that the specified finalizer is added to the given Kubernetes object within the provided timeout and interval.
func (k *KGinkgo) ExpectFinalizerToBeAdded(ctx context.Context, obj client.Object, finalizer string, opts ...KGinkgoOpt) {
	GinkgoHelper()

	eo := makeKGinkgoOptions(opts...)
	key := client.ObjectKeyFromObject(obj)

	Eventually(func(g Gomega) {
		fetched := &corev1.Service{}
		err := k.client.Get(ctx, key, fetched)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(fetched.GetFinalizers()).To(ContainElement(finalizer))
	}).WithTimeout(eo.timeout).WithPolling(eo.interval).Should(Succeed())
}

// ExpectFinalizerToExist asserts that the specified finalizer exists on the given Kubernetes object within the provided timeout and interval.
func (k *KGinkgo) ExpectFinalizerToBeRemoved(ctx context.Context, obj client.Object, finalizer string, opts ...KGinkgoOpt) {
	GinkgoHelper()

	eo := makeKGinkgoOptions(opts...)
	key := client.ObjectKeyFromObject(obj)

	Eventually(func(g Gomega) {
		fetched := &corev1.Service{}
		err := k.client.Get(ctx, key, fetched)

		// If the object is not found, the finalizer has been removed and the object deleted
		if client.IgnoreNotFound(err) == nil {
			return
		}

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(fetched.GetFinalizers()).ToNot(ContainElement(finalizer))
	}).WithTimeout(eo.timeout).WithPolling(eo.interval).Should(Succeed())
}

func (k *KGinkgo) ExpectHasAnnotation(ctx context.Context, obj client.Object, key string, opts ...KGinkgoOpt) {
	GinkgoHelper()

	k.EventuallyWithObject(ctx, obj, func(g Gomega, fetched client.Object) {
		annotations := fetched.GetAnnotations()
		g.Expect(annotations).NotTo(BeEmpty())

		g.Expect(annotations).To(HaveKey(key))
	}, opts...)
}

func (k *KGinkgo) ExpectAnnotationValue(ctx context.Context, obj client.Object, key, expectedValue string, opts ...KGinkgoOpt) {
	GinkgoHelper()

	k.EventuallyWithObject(ctx, obj, func(g Gomega, fetched client.Object) {
		annotations := fetched.GetAnnotations()
		g.Expect(annotations).NotTo(BeEmpty())

		actualValue, exists := annotations[key]
		g.Expect(exists).To(BeTrue(), "expected annotation %q to exist", key)
		g.Expect(actualValue).To(Equal(expectedValue), "expected annotation %q to have value %q but got %q", key, expectedValue, actualValue)
	}, opts...)
}

func (k *KGinkgo) EventuallyWithObject(ctx context.Context, obj client.Object, inner func(g Gomega, fetched client.Object), opts ...KGinkgoOpt) {
	GinkgoHelper()

	eo := makeKGinkgoOptions(opts...)
	objKey := client.ObjectKeyFromObject(obj)

	Eventually(func(g Gomega) {
		fetched := obj.DeepCopyObject().(client.Object)
		g.Expect(k.client.Get(ctx, objKey, fetched)).NotTo(HaveOccurred())

		inner(g, fetched)
	}).WithTimeout(eo.timeout).WithPolling(eo.interval).Should(Succeed())
}

func (k *KGinkgo) EventuallyWithCloudEndpoints(ctx context.Context, namespace string, inner func(g Gomega, cleps []ngrokv1alpha1.CloudEndpoint), opts ...KGinkgoOpt) {
	GinkgoHelper()
	eo := makeKGinkgoOptions(opts...)

	Eventually(func(g Gomega) {
		// List CloudEndpoints in the namespace
		cleps, err := k.getCloudEndpoints(ctx, namespace)
		g.Expect(err).NotTo(HaveOccurred())

		inner(g, cleps)
	}).WithTimeout(eo.timeout).WithPolling(eo.interval).Should(Succeed())
}

func (k *KGinkgo) ConsistentlyWithCloudEndpoints(ctx context.Context, namespace string, inner func(g Gomega, cleps []ngrokv1alpha1.CloudEndpoint), opts ...KGinkgoOpt) {
	GinkgoHelper()
	eo := makeKGinkgoOptions(opts...)

	Consistently(func(g Gomega) {
		// List CloudEndpoints in the namespace
		cleps, err := k.getCloudEndpoints(ctx, namespace)
		g.Expect(err).NotTo(HaveOccurred())

		inner(g, cleps)
	}).WithTimeout(eo.timeout).WithPolling(eo.interval).Should(Succeed())
}

func (k *KGinkgo) EventuallyWithAgentEndpoints(ctx context.Context, namespace string, inner func(g Gomega, aeps []ngrokv1alpha1.AgentEndpoint), opts ...KGinkgoOpt) {
	GinkgoHelper()
	eo := makeKGinkgoOptions(opts...)

	Eventually(func(g Gomega) {
		// List AgentEndpoints in the namespace
		aeps, err := k.getAgentEndpoints(ctx, namespace)
		g.Expect(err).NotTo(HaveOccurred())

		inner(g, aeps)
	}).WithTimeout(eo.timeout).WithPolling(eo.interval).Should(Succeed())
}

func (k *KGinkgo) EventuallyWithCloudAndAgentEndpoints(ctx context.Context, namespace string, inner func(g Gomega, cleps []ngrokv1alpha1.CloudEndpoint, aeps []ngrokv1alpha1.AgentEndpoint), opts ...KGinkgoOpt) {
	GinkgoHelper()
	eo := makeKGinkgoOptions(opts...)

	Eventually(func(g Gomega) {
		// List CloudEndpoints in the namespace
		cleps, err := k.getCloudEndpoints(ctx, namespace)
		g.Expect(err).NotTo(HaveOccurred())

		// List AgentEndpoints in the namespace
		aeps, err := k.getAgentEndpoints(ctx, namespace)
		g.Expect(err).NotTo(HaveOccurred())

		inner(g, cleps, aeps)
	}).WithTimeout(eo.timeout).WithPolling(eo.interval).Should(Succeed())
}

func (k *KGinkgo) EventuallyExpectNoEndpoints(ctx context.Context, namespace string, opts ...KGinkgoOpt) {
	GinkgoHelper()

	By("verifying no cloud or agent endpoints remain")
	k.EventuallyWithCloudAndAgentEndpoints(ctx, namespace, func(g Gomega, cleps []ngrokv1alpha1.CloudEndpoint, aeps []ngrokv1alpha1.AgentEndpoint) {
		By("verifying no cloud endpoints remain")
		g.Expect(cleps).To(BeEmpty())

		By("verifying no agent endpoints remain")
		g.Expect(aeps).To(BeEmpty())
	}, opts...)
}

func (k *KGinkgo) getCloudEndpoints(ctx context.Context, namespace string) ([]ngrokv1alpha1.CloudEndpoint, error) {
	GinkgoHelper()

	clepList := &ngrokv1alpha1.CloudEndpointList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
	}
	if err := k.client.List(ctx, clepList, listOpts...); err != nil {
		return nil, err
	}
	return clepList.Items, nil
}

func (k *KGinkgo) getAgentEndpoints(ctx context.Context, namespace string) ([]ngrokv1alpha1.AgentEndpoint, error) {
	GinkgoHelper()

	aepList := &ngrokv1alpha1.AgentEndpointList{}
	listOpts := []client.ListOption{
		client.InNamespace(namespace),
	}
	if err := k.client.List(ctx, aepList, listOpts...); err != nil {
		return nil, err
	}
	return aepList.Items, nil
}

func makeKGinkgoOptions(opts ...KGinkgoOpt) *expectOptions {
	eo := &expectOptions{
		timeout:  DefaultTimeout,
		interval: DefaultInterval,
	}
	for _, o := range opts {
		o(eo)
	}
	return eo
}
