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

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NgrokModuleSet is the Schema for the ngrokmodules API
type NgrokModuleSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Compression configuration for this module
	Compression *EndpointCompression `json:"compression,omitempty"`
	// Header configuration for this module
	Headers *EndpointHeaders `json:"headers,omitempty"`
	// IPRestriction configuration for this module
	IPRestriction *EndpointIPPolicy `json:"ipRestriction,omitempty"`
	// TLSTermination configuration for this module
	TLSTermination *EndpointTLSTerminationAtEdge `json:"tlsTermination,omitempty"`
	// WebhookVerification configuration for this module
	WebhookVerification *EndpointWebhookVerification `json:"webhookVerification,omitempty"`
}

//+kubebuilder:object:root=true

// NgrokModuleSetList contains a list of NgrokModule
type NgrokModuleSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NgrokModuleSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NgrokModuleSet{}, &NgrokModuleSetList{})
}
