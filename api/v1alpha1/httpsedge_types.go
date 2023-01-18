/*
MIT License

Copyright (c) 2022 ngrok, Inc.

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

type EndpointCompression struct {
	// Enabled is whether or not to enable compression for this endpoint
	Enabled *bool `json:"enabled,omitempty"`
}
type HTTPSEdgeRouteSpec struct {
	ngrokAPICommon `json:",inline"`

	// MatchType is the type of match to use for this route. Valid values are:
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=exact_path;path_prefix
	MatchType string `json:"matchType"`

	// Match is the value to match against the request path
	// +kubebuilder:validation:Required
	Match string `json:"match"`

	// Backend is the definition for the tunnel group backend
	// that serves traffic for this edge
	// +kubebuilder:validation:Required
	Backend TunnelGroupBackend `json:"backend,omitempty"`

	// Compression is whether or not to enable compression for this route
	Compression *EndpointCompression `json:"compression,omitempty"`
}

// HTTPSEdgeSpec defines the desired state of HTTPSEdge
type HTTPSEdgeSpec struct {
	ngrokAPICommon `json:",inline"`

	// Hostports is a list of hostports served by this edge
	// +kubebuilder:validation:Required
	Hostports []string `json:"hostports,omitempty"`

	// Routes is a list of routes served by this edge
	Routes []HTTPSEdgeRouteSpec `json:"routes,omitempty"`
}

type HTTPSEdgeRouteStatus struct {
	// ID is the unique identifier for this route
	ID string `json:"id,omitempty"`

	// URI is the URI for this route
	URI string `json:"uri,omitempty"`

	Match string `json:"match,omitempty"`

	MatchType string `json:"matchType,omitempty"`

	// Backend stores the status of the tunnel group backend,
	// mainly the ID of the backend
	Backend TunnelGroupBackendStatus `json:"backend,omitempty"`
}

// HTTPSEdgeStatus defines the observed state of HTTPSEdge
type HTTPSEdgeStatus struct {
	// ID is the unique identifier for this edge
	ID string `json:"id,omitempty"`

	// URI is the URI for this edge
	URI string `json:"uri,omitempty"`

	Routes []HTTPSEdgeRouteStatus `json:"routes,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// HTTPSEdge is the Schema for the httpsedges API
type HTTPSEdge struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HTTPSEdgeSpec   `json:"spec,omitempty"`
	Status HTTPSEdgeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HTTPSEdgeList contains a list of HTTPSEdge
type HTTPSEdgeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HTTPSEdge `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HTTPSEdge{}, &HTTPSEdgeList{})
}