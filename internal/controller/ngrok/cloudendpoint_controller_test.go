package ngrok

import (
	"encoding/json"
	"testing"

	"github.com/go-logr/logr"
	ingressv1alpha1 "github.com/ngrok/ngrok-operator/api/ingress/v1alpha1"
	ngrokv1alpha1 "github.com/ngrok/ngrok-operator/api/ngrok/v1alpha1"
	"github.com/ngrok/ngrok-operator/internal/services"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_findTrafficPolicy(t *testing.T) {
	// Set up a fake client with a sample TrafficPolicy
	scheme := runtime.NewScheme()
	_ = ngrokv1alpha1.AddToScheme(scheme)

	rawPolicy := json.RawMessage(`{"type": "allow"}`)

	trafficPolicy := &ngrokv1alpha1.NgrokTrafficPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "policy-1",
			Namespace: "default",
		},
		Spec: ngrokv1alpha1.NgrokTrafficPolicySpec{
			Policy: rawPolicy,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(trafficPolicy).
		Build()

	r := &CloudEndpointReconciler{
		Client:   fakeClient,
		Recorder: record.NewFakeRecorder(10),
	}

	// Call the function under test
	policy, err := r.findTrafficPolicyByName(t.Context(), "policy-1", "default")

	// Assert that the correct policy is found
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"allow"}`, policy)

	// Test case where TrafficPolicy is not found
	policy, err = r.findTrafficPolicyByName(t.Context(), "nonexistent-policy", "default")
	assert.Error(t, err)
	assert.Equal(t, "", policy)
}

func Test_ensureDomainExists(t *testing.T) {
	// Set up a fake client with a sample Domain
	scheme := runtime.NewScheme()
	assert.NoError(t, ingressv1alpha1.AddToScheme(scheme))
	assert.NoError(t, ngrokv1alpha1.AddToScheme(scheme))

	existingNotReadyDomain := &ingressv1alpha1.Domain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-com",
			Namespace: "default",
		},
	}
	existingReadyDomain := &ingressv1alpha1.Domain{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example2-com",
			Namespace: "default",
		},
		Spec: ingressv1alpha1.DomainSpec{
			Domain: "example2.com",
		},
		Status: ingressv1alpha1.DomainStatus{
			ID: "rd_123",
		},
	}
	clep1 := &ngrokv1alpha1.CloudEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloud-endpoint-1",
			Namespace: "default",
		},
		Spec: ngrokv1alpha1.CloudEndpointSpec{
			URL: "https://example.com",
		},
	}
	clep2 := &ngrokv1alpha1.CloudEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloud-endpoint-2",
			Namespace: "default",
		},
		Spec: ngrokv1alpha1.CloudEndpointSpec{
			URL: "https://example2.com",
		},
	}
	clep3 := &ngrokv1alpha1.CloudEndpoint{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloud-endpoint-3",
			Namespace: "default",
		},
		Spec: ngrokv1alpha1.CloudEndpointSpec{
			URL: "https://newdomain.com",
		},
	}

	objs := []client.Object{
		existingNotReadyDomain,
		existingReadyDomain,
		clep1,
		clep2,
		clep3,
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&ngrokv1alpha1.CloudEndpoint{}).
		Build()

	r := &CloudEndpointReconciler{
		Client:        fakeClient,
		Log:           logr.Discard(),
		Recorder:      record.NewFakeRecorder(10),
		domainService: services.NewDefaultDomainService(fakeClient),
	}

	// Case 1: Domain already exists, but is not ready
	err := r.ensureDomainOrAddrExists(t.Context(), clep1)
	assert.Equal(t, ErrDomainCreating, err)
	assert.Empty(t, clep1.Status.Domain)

	// Case 2: Domain already exists and is ready
	err = r.ensureDomainOrAddrExists(t.Context(), clep2)
	assert.NoError(t, err)
	// The cloudendpoint's status should have the domain status
	assert.NotNil(t, clep2.Status.Domain)
	assert.Equal(t, "rd_123", clep2.Status.Domain.ID)

	// Case 3: Domain does not exist and should be created
	err = r.ensureDomainOrAddrExists(t.Context(), clep3)
	assert.Equal(t, ErrDomainCreating, err)
	assert.Empty(t, clep3.Status.Domain)
}
