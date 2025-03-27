package services

import (
	"testing"

	ingressv1alpha1 "github.com/ngrok/ngrok-operator/api/ingress/v1alpha1"
	ngrokv1alpha1 "github.com/ngrok/ngrok-operator/api/ngrok/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDefaultDomainServiceImplementsDomainService(t *testing.T) {
	assert.Implements(t, (*DomainService)(nil), &DefaultDomainService{})
}

var _ = Describe("DefaultDomainService", func() {
	var (
		namespace = "default"
		service   DomainService
		k8sClient client.Client
		objs      []runtime.Object
		scheme    *runtime.Scheme
	)

	JustBeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(ingressv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(ngrokv1alpha1.AddToScheme(scheme)).To(Succeed())

		k8sClient = fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(objs...).
			WithStatusSubresource(&ingressv1alpha1.Domain{}).
			Build()
		service = NewDefaultDomainService(k8sClient)
	})

	Describe("FindOrReserveDomain", func() {
		var (
			domainString string
			forObj       client.Object
			domain       *ingressv1alpha1.Domain
			err          error
		)

		JustBeforeEach(func() {
			domain, err = service.FindOrReserveDomain(GinkgoT().Context(), forObj, domainString)
		})

		BeforeEach(func() {
			forObj = &ngrokv1alpha1.CloudEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-endpoint",
					Namespace: namespace,
				},
				Spec: ngrokv1alpha1.CloudEndpointSpec{
					URL: "https://test-1234.example.com",
				},
			}
		})

		When("The domain CR does not exist yet", func() {
			BeforeEach(func() {
				domainString = "test-not-exist.reserve-domain.com"
			})

			It("Should create a new domain CR", func() {
				Expect(domain).ToNot(BeNil())
				Expect(domain.Name).To(Equal("test-not-exist-reserve-domain-com"))
			})

			It("Should return an ErrDomainCreating error", func() {
				Expect(err).To(Equal(ErrDomainCreating))
			})
		})

		When("The domain is an internal domain", func() {
			BeforeEach(func() {
				domainString = "something.other.internal"
			})

			It("Should not create a domain", func() {
				Expect(err).To(BeNil())
			})

			It("Should not return an error", func() {
				Expect(domain).To(BeNil())
			})
		})

		When("The forObj is nil", func() {
			BeforeEach(func() {
				forObj = nil
			})

			It("Should return ErrNoObjectForDomain", func() {
				Expect(err).To(Equal(ErrNoObjectForDomain))
			})

			It("Should not return a domain", func() {
				Expect(domain).To(BeNil())
			})
		})

		When("The domain exists", func() {
			BeforeEach(func() {
				domainString = "a-domain.that-exists.com"
				objs = []runtime.Object{
					&ingressv1alpha1.Domain{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "a-domain-that-exists-com",
							Namespace: namespace,
						},
						Status: ingressv1alpha1.DomainStatus{},
					},
				}
			})

			When("The domain is not ready", func() {
				It("Should return the domain", func() {
					Expect(domain).ToNot(BeNil())
					Expect(domain.Name).To(Equal("a-domain-that-exists-com"))
				})

				It("Should return an ErrDomainCreating error", func() {
					Expect(err).To(Equal(ErrDomainCreating))
				})
			})

			When("The domain is ready", func() {
				BeforeEach(func() {
					objs[0].(*ingressv1alpha1.Domain).Status.ID = "rdd_123456789"
				})

				It("Should return the domain", func() {
					Expect(domain).ToNot(BeNil())
					Expect(domain.Name).To(Equal("a-domain-that-exists-com"))
					Expect(domain.Status.ID).To(Equal("rdd_123456789"))
				})

				It("Should not return an error", func() {
					Expect(err).To(BeNil())
				})
			})
		})
	})

})
