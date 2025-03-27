package services

import (
	"context"
	"errors"
	"strings"

	ingressv1alpha1 "github.com/ngrok/ngrok-operator/api/ingress/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrDomainCreating    = errors.New("domain is being created, requeue after delay")
	ErrNoObjectForDomain = errors.New("no object provided to reserve domain for")
)

type DomainService interface {
	FindOrReserveDomain(context.Context, client.Object, string) (*ingressv1alpha1.Domain, error)
}

type DefaultDomainService struct {
	k8sClient client.Client
}

func NewDefaultDomainService(k8sClient client.Client) *DefaultDomainService {
	return &DefaultDomainService{
		k8sClient: k8sClient,
	}
}

// FindOrReserveDomain finds or creates a new Domain CR for the given object. It will return the
// domain CR if it already exists, or create a new one if it does not. It will return an error if
// it is unable to create the domain CR or if the Domain CR is not finished reserving.
func (r DefaultDomainService) FindOrReserveDomain(ctx context.Context, forObj client.Object, domain string) (*ingressv1alpha1.Domain, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("domain", domain)

	if forObj == nil {
		log.V(5).Info("No object provided to reserve domain for")
		return nil, ErrNoObjectForDomain
	}

	if strings.HasSuffix(domain, ".internal") {
		log.V(5).Info("Skipping reserving an internal domain")
		return nil, nil
	}

	hyphenatedDomain := ingressv1alpha1.HyphenatedDomainNameFromURL(domain)

	domainObj := &ingressv1alpha1.Domain{}

	// Try to get directly by key
	objKey := client.ObjectKey{Namespace: forObj.GetNamespace(), Name: hyphenatedDomain}
	err := r.k8sClient.Get(ctx, objKey, domainObj)
	if err == nil {
		// Domain already exists
		if domainObj.Status.ID == "" {
			// Domain is not ready yet
			return domainObj, ErrDomainCreating
		}
		// Domain is ready
		return domainObj, nil
	}

	if client.IgnoreNotFound(err) != nil {
		// Some other error occurred
		log.Error(err, "failed to check Domain CRD existence")
		return domainObj, err
	}

	// Fallback to searching if the name isn't well known
	domains := &ingressv1alpha1.DomainList{}
	opts := []client.ListOption{
		client.InNamespace(forObj.GetNamespace()),
	}
	if err := r.k8sClient.List(ctx, domains, opts...); err != nil {
		log.Error(err, "failed to list Domain CRDs")
		return nil, err
	}

	for _, d := range domains.Items {
		if d.Spec.Domain == domain {
			// Found a matching domain
			if d.Status.ID == "" {
				// Domain is not ready yet
				return &d, ErrDomainCreating
			}
		}
	}

	// Create the Domain CR since it doesn't exist
	newDomain := &ingressv1alpha1.Domain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hyphenatedDomain,
			Namespace: forObj.GetNamespace(),
		},
		Spec: ingressv1alpha1.DomainSpec{
			Domain: domain,
		},
	}
	if err := r.k8sClient.Create(ctx, newDomain); err != nil {
		return newDomain, err
	}

	// We create the new domain, but it's not ready yet
	return newDomain, ErrDomainCreating
}
