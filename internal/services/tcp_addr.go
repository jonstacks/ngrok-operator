package services

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ngrok/ngrok-api-go/v7"
	"github.com/ngrok/ngrok-operator/internal/ngrokapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TCPAddrService interface {
	FindOrReserveAddr(context context.Context, forObj client.Object) (*ngrok.ReservedAddr, error)
}

type DefaultTCPAddrService struct {
	client ngrokapi.TCPAddressesClient
}

func NewDefaultTCPAddrService(client ngrokapi.TCPAddressesClient) *DefaultTCPAddrService {
	return &DefaultTCPAddrService{
		client: client,
	}
}

func (r DefaultTCPAddrService) FindOrReserveAddr(ctx context.Context, forObj client.Object) (*ngrok.ReservedAddr, error) {
	log := ctrl.LoggerFrom(ctx)

	metadata := reservedAddrMetadata{
		Namespace: forObj.GetNamespace(),
		Name:      forObj.GetName(),
		OwnerRef:  metav1.GetControllerOf(forObj),
	}

	log = log.WithValues("metadata", metadata)

	addr, err := r.findAddrWithMatchingMetadata(ctx, metadata)
	if err != nil {
		log.Error(err, "Failed to iterate over existing reserved addresses")
		return nil, err
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	// If we found an addr with matching metadata, use it. Update the addr description & metadata.
	// We know the metadata matches, but the name for may have changed.
	if addr != nil {
		log = log.WithValues("reservedAddr.ID", addr.ID, "reservedAddr.Addr", addr.Addr)

		log.V(3).Info("Found existing addr with matching metadata")
		description := fmt.Sprintf("Reserved for %s/%s", forObj.GetNamespace(), forObj.GetName())
		metadata := string(metadataBytes)

		addr, err = r.client.Update(ctx, &ngrok.ReservedAddrUpdate{
			ID:          addr.ID,
			Description: &description,
			Metadata:    &metadata,
		})
		if err != nil {
			log.Error(err, "Failed to update reserved address")
			return nil, err
		}
		return addr, err
	}
	log.V(3).Info("Creating new reserved address")
	return r.client.Create(ctx, &ngrok.ReservedAddrCreate{
		Description: fmt.Sprintf("Reserved for %s/%s", forObj.GetNamespace(), forObj.GetName()),
		Metadata:    string(metadataBytes),
	})
}

// findAddrWithMatchingMetadata finds a reserved address with the same metadata as the given metadata.
// Returns nil if no matching address is found. Will return an error if there is an issue listing the addresses.
func (r DefaultTCPAddrService) findAddrWithMatchingMetadata(ctx context.Context, metadata reservedAddrMetadata) (*ngrok.ReservedAddr, error) {
	log := ctrl.LoggerFrom(ctx)

	iter := r.client.List(&ngrok.Paging{})
	for iter.Next(ctx) {
		addr := iter.Item()
		if addr.Metadata == "" {
			continue
		}

		addrMetadata := reservedAddrMetadata{}
		// if we can't unmarshal the metadata, we can't match it, but just continue
		if err := json.Unmarshal([]byte(addr.Metadata), &addrMetadata); err != nil {
			log.V(3).Info("Failed to unmarshal addr metadata", "addr.ID", addr.ID, "err", err)
			continue
		}

		if metadata.Matches(addrMetadata) {
			return addr, nil
		}
	}
	return nil, iter.Err()
}

type reservedAddrMetadata struct {
	Namespace string                 `json:"namespace"`
	Name      string                 `json:"name"`
	OwnerRef  *metav1.OwnerReference `json:"ownerRef,omitempty"`
}

// Matches returns true if the metadata is a match for the other metadata
func (m reservedAddrMetadata) Matches(other reservedAddrMetadata) bool {
	// If the namespaces don't match, they're automatically not a match
	if m.Namespace != other.Namespace {
		return false
	}

	// If either have the owner reference set, we'll use those to compare
	if m.OwnerRef != nil || other.OwnerRef != nil {
		return reflect.DeepEqual(m.OwnerRef, other.OwnerRef)
	}

	return m.Name == other.Name
}
