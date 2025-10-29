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
package agent

import (
	ngrokv1alpha1 "github.com/ngrok/ngrok-operator/api/ngrok/v1alpha1"
)

// Type aliases
type (
	AgentEndpoint                  = ngrokv1alpha1.AgentEndpoint
	AgentEndpointList              = ngrokv1alpha1.AgentEndpointList
	AgentEndpointSpec              = ngrokv1alpha1.AgentEndpointSpec
	ConditionType                  = ngrokv1alpha1.AgentEndpointConditionType
	ConditionReadyReason           = ngrokv1alpha1.AgentEndpointConditionReadyReason
	ConditionEndpointCreatedReason = ngrokv1alpha1.AgentEndpointConditionEndpointCreatedReason
	ConditionTrafficPolicyReason   = ngrokv1alpha1.AgentEndpointConditionTrafficPolicyReason
	EndpointUpstream               = ngrokv1alpha1.EndpointUpstream
	K8sObjectRef                   = ngrokv1alpha1.K8sObjectRef
	K8sObjectRefOptionalNamespace  = ngrokv1alpha1.K8sObjectRefOptionalNamespace
	NgrokTrafficPolicy             = ngrokv1alpha1.NgrokTrafficPolicy
	NgrokTrafficPolicySpec         = ngrokv1alpha1.NgrokTrafficPolicySpec
	TrafficPolicyCfg               = ngrokv1alpha1.TrafficPolicyCfg
)

// Condition aliases
const (
	ConditionEndpointCreated = ngrokv1alpha1.AgentEndpointConditionEndpointCreated
	ConditionReady           = ngrokv1alpha1.AgentEndpointConditionReady
	ConditionTrafficPolicy   = ngrokv1alpha1.AgentEndpointConditionTrafficPolicy
)

// Reason aliases
const (
	ReasonActive               = ngrokv1alpha1.AgentEndpointReasonActive
	ReasonConfigError          = ngrokv1alpha1.AgentEndpointReasonConfigError
	ReasonDomainNotReady       = ngrokv1alpha1.AgentEndpointReasonDomainNotReady
	ReasonEndpointCreated      = ngrokv1alpha1.AgentEndpointReasonEndpointCreated
	ReasonNgrokAPIError        = ngrokv1alpha1.AgentEndpointReasonNgrokAPIError
	ReasonPending              = ngrokv1alpha1.AgentEndpointReasonPending
	ReasonReconciling          = ngrokv1alpha1.AgentEndpointReasonReconciling
	ReasonTrafficPolicyApplied = ngrokv1alpha1.AgentEndpointReasonTrafficPolicyApplied
	ReasonTrafficPolicyError   = ngrokv1alpha1.AgentEndpointReasonTrafficPolicyError
	ReasonUnknown              = ngrokv1alpha1.AgentEndpointReasonUnknown
)

// TrafficPolicy aliases
const (
	TrafficPolicyCfgType_Inline = ngrokv1alpha1.TrafficPolicyCfgType_Inline
	TrafficPolicyCfgType_K8sRef = ngrokv1alpha1.TrafficPolicyCfgType_K8sRef
)
