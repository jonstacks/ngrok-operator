/*
MIT License

Copyright (c) 2024 ngrok, Inc.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type KubernetesOperatorDeployment struct {
	// name is the name of the k8s deployment for the operator
	// +optional
	Name string `json:"name,omitempty"`
	// namespace is the namespace in which the operator is deployed
	// +optional
	Namespace string `json:"namespace,omitempty"`
	// version is the version of the operator that is currently running
	// +optional
	Version string `json:"version,omitempty"`
}

type KubernetesOperatorBinding struct {
	// endpointSelectors is a list of cel expression that determine which kubernetes-bound Endpoints will be created by the operator
	// +required
	EndpointSelectors []string `json:"endpointSelectors,omitempty"`

	// ingressEndpoint is the public ingress endpoint for this Kubernetes Operator
	// +optional
	IngressEndpoint *string `json:"ingressEndpoint,omitempty"`

	// tlsSecretName is the name of the k8s secret that contains the TLS private/public keys to use for the ngrok forwarding endpoint
	// +kubebuilder:default="default-tls"
	// +optional
	TlsSecretName string `json:"tlsSecretName"`
}

// KubernetesOperatorStatus defines the observed state of KubernetesOperator
type KubernetesOperatorStatus struct {
	// id is the unique identifier for this Kubernetes Operator
	// +optional
	ID string `json:"id,omitempty"`

	// uri is the URI for this Kubernetes Operator
	// +optional
	URI string `json:"uri,omitempty"`

	// registrationStatus is the status of the registration of this Kubernetes Operator with the ngrok API
	// +kubebuilder:validation:Enum=registered;error;pending
	// +default="pending"
	RegistrationStatus string `json:"registrationStatus,omitempty"`

	// registrationErrorCode is the returned ngrok error code
	// +kubebuilder:validation:Pattern=`^ERR_NGROK_\d+$`
	// +optional
	RegistrationErrorCode string `json:"registrationErrorCode,omitempty"`

	// errorMessage is a free-form error message if the status is error
	// +kubebuilder:validation:MaxLength=4096
	// +optional
	RegistrationErrorMessage string `json:"errorMessage,omitempty"`

	// enabledFeatures is the string representation of the features enabled for this Kubernetes Operator
	// +optional
	EnabledFeatures string `json:"enabledFeatures,omitempty"`

	// bindingsIngressEndpoint is the URL that the operator will use to talk
	// to the ngrok edge when forwarding traffic for k8s-bound endpoints
	// +optional
	BindingsIngressEndpoint string `json:"bindingsIngressEndpoint,omitempty"`

	// drainStatus indicates the current state of the drain process
	// +optional
	DrainStatus DrainStatus `json:"drainStatus,omitempty"`

	// drainMessage provides additional information about the drain status
	// +optional
	DrainMessage string `json:"drainMessage,omitempty"`

	// drainProgress indicates how many resources have been drained vs total
	// Format: "X/Y" where X is processed (completed + failed) and Y is total
	// +optional
	DrainProgress string `json:"drainProgress,omitempty"`

	// drainErrors contains the most recent errors encountered during drain
	// +optional
	// +listType=atomic
	DrainErrors []string `json:"drainErrors,omitempty"`
}

const (
	KubernetesOperatorRegistrationStatusSuccess = "registered"
	KubernetesOperatorRegistrationStatusError   = "error"
	KubernetesOperatorRegistrationStatusPending = "pending"
)

const (
	KubernetesOperatorFeatureIngress  = "ingress"
	KubernetesOperatorFeatureGateway  = "gateway"
	KubernetesOperatorFeatureBindings = "bindings"
)

// DrainStatus indicates the current state of the drain process
// +kubebuilder:validation:Enum=pending;draining;completed;failed
type DrainStatus string

const (
	DrainStatusPending   DrainStatus = "pending"
	DrainStatusDraining  DrainStatus = "draining"
	DrainStatusCompleted DrainStatus = "completed"
	DrainStatusFailed    DrainStatus = "failed"
)

// DrainPolicy determines how ngrok API resources are handled during drain
// +kubebuilder:validation:Enum=Delete;Retain
type DrainPolicy string

const (
	// DrainPolicyDelete deletes the CR, triggering controllers to clean up ngrok API resources
	DrainPolicyDelete DrainPolicy = "Delete"
	// DrainPolicyRetain removes finalizers but preserves ngrok API resources
	DrainPolicyRetain DrainPolicy = "Retain"
)

type KubernetesOperatorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// description is a human-readable description of the object in the ngrok API/Dashboard
	// +default="Created by ngrok-operator"
	Description string `json:"description,omitempty"`

	// metadata is a string of arbitrary data associated with the object in the ngrok API/Dashboard
	// +default="{\"owned-by\":\"ngrok-operator\"}"
	Metadata string `json:"metadata,omitempty"`

	// enabledFeatures is the list of features enabled for this Kubernetes Operator
	// +kubebuilder:validation:items:Enum=ingress;gateway;bindings
	// +optional
	EnabledFeatures []string `json:"enabledFeatures,omitempty"`

	// region is the ngrok region in which the ingress for this operator is served. Defaults to
	// "global" if not specified.
	// +default="global"
	Region string `json:"region,omitempty"`

	// deployment contains information of this Kubernetes Operator
	// +optional
	Deployment *KubernetesOperatorDeployment `json:"deployment,omitempty"`

	// binding is the configuration for the binding feature of this Kubernetes Operator
	// +optional
	Binding *KubernetesOperatorBinding `json:"binding,omitempty"`

	// drain configures the drain behavior for uninstall
	// +optional
	Drain *DrainConfig `json:"drain,omitempty"`
}

// DrainConfig configures the drain behavior during operator uninstall
type DrainConfig struct {
	// policy determines whether to delete ngrok API resources or just remove finalizers
	// +default="Retain"
	Policy DrainPolicy `json:"policy,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="ID",type=string,JSONPath=`.status.id`,description="Kubernetes Operator ID"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.registrationStatus"
// +kubebuilder:printcolumn:name="Enabled Features",type="string",JSONPath=".status.enabledFeatures"
// +kubebuilder:printcolumn:name="Endpoint Selectors",type="string",JSONPath=".spec.binding.endpointSelectors"
// +kubebuilder:printcolumn:name="Binding Ingress Endpoint", type="string", JSONPath=".spec.binding.ingressEndpoint",priority=2
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`,description="Age"

// KubernetesOperator is the Schema for the ngrok kubernetesoperators API
type KubernetesOperator struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is the standard object metadata.
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of KubernetesOperator.
	Spec KubernetesOperatorSpec `json:"spec,omitempty"`
	// status defines the observed state of KubernetesOperator.
	Status KubernetesOperatorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KubernetesOperatorList contains a list of KubernetesOperator
type KubernetesOperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubernetesOperator `json:"items"`
}

// GetDrainPolicy returns the configured drain policy, defaulting to Retain if not set.
func (ko *KubernetesOperator) GetDrainPolicy() DrainPolicy {
	if ko.Spec.Drain != nil && ko.Spec.Drain.Policy != "" {
		return ko.Spec.Drain.Policy
	}
	return DrainPolicyRetain
}

func init() {
	SchemeBuilder.Register(&KubernetesOperator{}, &KubernetesOperatorList{})
}
