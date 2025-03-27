package services

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ngrok/ngrok-api-go/v7"
	ngrokv1alpha1 "github.com/ngrok/ngrok-operator/api/ngrok/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestDefaultTCPAddrServiceImplementsTCPAddrService(t *testing.T) {
	assert.Implements(t, (*TCPAddrService)(nil), &DefaultTCPAddrService{})
}

func TestReservedAddrMetadataMatches(t *testing.T) {
	tests := []struct {
		name string
		m1   reservedAddrMetadata
		m2   reservedAddrMetadata
		want bool
	}{
		{
			name: "both empty",
			m1:   reservedAddrMetadata{},
			m2:   reservedAddrMetadata{},
			want: true,
		},
		{
			name: "Namespace mismatch",
			m1:   reservedAddrMetadata{Name: "one", Namespace: "namespace-1", OwnerRef: nil},
			m2:   reservedAddrMetadata{Name: "two", Namespace: "namespace-2", OwnerRef: nil},
			want: false,
		},
		{
			name: "OwnerRef not present on one",
			m1:   reservedAddrMetadata{Name: "one", Namespace: "namespace-1", OwnerRef: nil},
			m2:   reservedAddrMetadata{Name: "two", Namespace: "namespace-1", OwnerRef: &metav1.OwnerReference{Kind: "kind", Name: "name"}},
			want: false,
		},
		{
			name: "name mismatch with nil OwnerRefs",
			m1:   reservedAddrMetadata{Name: "one", Namespace: "namespace-1", OwnerRef: nil},
			m2:   reservedAddrMetadata{Name: "two", Namespace: "namespace-1", OwnerRef: nil},
			want: false,
		},
		{
			name: "name mismatch with matching OwnerRefs",
			m1:   reservedAddrMetadata{Name: "one", Namespace: "namespace-1", OwnerRef: &metav1.OwnerReference{Kind: "kind", Name: "name-unique"}},
			m2:   reservedAddrMetadata{Name: "two", Namespace: "namespace-1", OwnerRef: &metav1.OwnerReference{Kind: "kind", Name: "name-unique"}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.m1.Matches(tt.m2)
			assert.Equal(t, tt.want, got)
		})
	}
}

type MockIter[T any] struct {
	items    []T
	err      error
	curIndex int
}

func newMockIter[T any](items []T) *MockIter[T] {
	if items == nil {
		items = []T{}
	}
	return &MockIter[T]{
		items:    items,
		curIndex: -1,
	}
}

func (m *MockIter[T]) Next(ctx context.Context) bool {
	if m.curIndex+1 < len(m.items) {
		m.curIndex++
		return true
	}
	return false
}

func (m *MockIter[T]) Item() T {
	return m.items[m.curIndex]
}

func (m *MockIter[T]) Err() error {
	return m.err
}

type MockTCPAddressesClientCreateResp struct {
	ID   string
	Addr string
}

// TODO(stacks): This mock is probably more useful in a shared package for other testing. I'm
// still working out the API for the mocks, so we'll see how this evolves for making testing easier.
type MockTCPAddressesClient struct {
	items []*ngrok.ReservedAddr

	createResp       MockTCPAddressesClientCreateResp
	createCalledWith *ngrok.ReservedAddrCreate
	createErr        error

	updateCalledWith *ngrok.ReservedAddrUpdate
	updateResp       *ngrok.ReservedAddr
	updateErr        error

	listCalled bool
}

func (m *MockTCPAddressesClient) Create(ctx context.Context, addr *ngrok.ReservedAddrCreate) (*ngrok.ReservedAddr, error) {
	m.createCalledWith = addr

	if m.createErr != nil {
		return nil, m.createErr
	}

	// Otherwise return a mock response
	return &ngrok.ReservedAddr{
		ID:          m.createResp.ID,
		Addr:        m.createResp.Addr,
		Description: addr.Description,
		Metadata:    addr.Metadata,
		Region:      "",
	}, m.createErr
}

func (m *MockTCPAddressesClient) List(paging *ngrok.Paging) ngrok.Iter[*ngrok.ReservedAddr] {
	m.listCalled = true
	if m.items == nil {
		m.items = []*ngrok.ReservedAddr{}
	}
	return newMockIter(m.items)
}

func (m *MockTCPAddressesClient) Update(ctx context.Context, addr *ngrok.ReservedAddrUpdate) (*ngrok.ReservedAddr, error) {
	m.updateCalledWith = addr

	// Return the mock response if it's set
	if m.updateResp != nil || m.updateErr != nil {
		return m.updateResp, m.updateErr
	}

	// Try to find the address in the items and use that to fill in the rest of the data
	for _, item := range m.items {
		if item.ID == addr.ID {
			// Update the item in the list
			if addr.Description != nil {
				item.Description = *addr.Description
			}
			if addr.Metadata != nil {
				item.Metadata = *addr.Metadata
			}
			return item, nil
		}
	}

	return nil, fmt.Errorf("address with ID %s not found", addr.ID)
}

var _ = Describe("DefaultTCPAddrService", func() {
	var (
		tcpAddrClient *MockTCPAddressesClient
		service       TCPAddrService
		namespace     = "test-tcp-addr-namespace"
	)

	BeforeEach(func() {
		tcpAddrClient = &MockTCPAddressesClient{}
		service = NewDefaultTCPAddrService(tcpAddrClient)
	})

	Describe("FindOrReserveAddr", func() {
		var (
			addr   *ngrok.ReservedAddr
			err    error
			forObj client.Object
		)

		JustBeforeEach(func() {
			addr, err = service.FindOrReserveAddr(GinkgoT().Context(), forObj)
		})

		BeforeEach(func() {
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-lb-service",
					Namespace: namespace,
				},
				Spec: corev1.ServiceSpec{},
			}

			forObj = &ngrokv1alpha1.CloudEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-endpoint",
					Namespace: namespace,
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(svc, corev1.SchemeGroupVersion.WithKind("Service")),
					},
				},
				Spec: ngrokv1alpha1.CloudEndpointSpec{
					URL: "tcp://",
				},
			}
		})

		When("No matching reserved address exists in ngrok", func() {
			BeforeEach(func() {
				tcpAddrClient.createResp = MockTCPAddressesClientCreateResp{
					ID:   "rd_123",
					Addr: "1.ngrok.io:1234",
				}
			})

			It("Should try to list the existing reserved addresses", func() {
				Expect(tcpAddrClient.listCalled).To(BeTrue())
			})

			It("Should call the ngrok API to create a new reserved address", func() {
				Expect(tcpAddrClient.createCalledWith).ToNot(BeNil())
				Expect(tcpAddrClient.createCalledWith.Description).To(Equal("Reserved for test-tcp-addr-namespace/cloud-endpoint"))
				Expect(tcpAddrClient.createCalledWith.Metadata).ToNot(BeEmpty())
			})

			It("Should create a new reserved address", func() {
				Expect(err).To(BeNil())
				Expect(addr).ToNot(BeNil())
				Expect(addr.ID).To(Equal("rd_123"))
				Expect(addr.Addr).To(Equal("1.ngrok.io:1234"))
			})
		})

		When("able to match a reserved address with similar metadata", func() {
			BeforeEach(func() {
				expectedMetadata := reservedAddrMetadata{
					Namespace: namespace,
					Name:      "cloud-endpoint",
					OwnerRef:  metav1.GetControllerOf(forObj),
				}
				expectedMetadataBytes, err := json.Marshal(expectedMetadata)
				Expect(err).To(BeNil())

				tcpAddrClient.items = []*ngrok.ReservedAddr{
					{
						ID:          "rd_456",
						Addr:        "1.ngrok.io:2222",
						Description: "Reserved for test-tcp-addr-namespace/cloud-endpoint",
						Metadata:    string(expectedMetadataBytes),
					},
				}
			})

			It("Should not call the ngrok API to create a new reserved address", func() {
				Expect(tcpAddrClient.createCalledWith).To(BeNil())
			})

			It("Should not return an error", func() {
				Expect(err).To(BeNil())
			})

			It("Should return the existing reserved address", func() {
				Expect(addr).ToNot(BeNil())
				Expect(addr.ID).To(Equal("rd_456"))
				Expect(addr.Addr).To(Equal("1.ngrok.io:2222"))
			})
		})
	})
})
