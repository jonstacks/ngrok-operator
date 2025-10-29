package agent

import (
	domainpkg "github.com/ngrok/ngrok-operator/internal/domain"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// setReadyCondition sets the Ready condition based on the overall endpoint state
func setReadyCondition(endpoint *AgentEndpoint, ready bool, reason ConditionReadyReason, message string) {
	status := metav1.ConditionTrue
	if !ready {
		status = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               string(ConditionReady),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: endpoint.Generation,
	}

	meta.SetStatusCondition(&endpoint.Status.Conditions, condition)
}

// setEndpointCreatedCondition sets the EndpointCreated condition
func setEndpointCreatedCondition(endpoint *AgentEndpoint, created bool, reason ConditionEndpointCreatedReason, message string) {
	status := metav1.ConditionTrue
	if !created {
		status = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               string(ConditionEndpointCreated),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: endpoint.Generation,
	}

	meta.SetStatusCondition(&endpoint.Status.Conditions, condition)
}

// setTrafficPolicyCondition sets the TrafficPolicyApplied condition
func setTrafficPolicyCondition(endpoint *AgentEndpoint, applied bool, reason ConditionTrafficPolicyReason, message string) {
	status := metav1.ConditionTrue
	if !applied {
		status = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               string(ConditionTrafficPolicy),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: endpoint.Generation,
	}

	meta.SetStatusCondition(&endpoint.Status.Conditions, condition)
}

// setReconcilingCondition sets a temporary reconciling condition
func setReconcilingCondition(endpoint *AgentEndpoint, message string) {
	setReadyCondition(endpoint, false, ReasonReconciling, message)
}

// calculateAgentEndpointReadyCondition calculates the overall Ready condition based on other conditions and domain status
func calculateAgentEndpointReadyCondition(aep *AgentEndpoint, domainResult *domainpkg.DomainResult) {
	// Check all required conditions
	endpointCreatedCondition := meta.FindStatusCondition(aep.Status.Conditions, string(ConditionEndpointCreated))
	endpointCreated := endpointCreatedCondition != nil && endpointCreatedCondition.Status == metav1.ConditionTrue

	trafficPolicyCondition := meta.FindStatusCondition(aep.Status.Conditions, string(ConditionTrafficPolicy))
	trafficPolicyReady := true
	// If traffic policy condition exists and is False, it's not ready
	if trafficPolicyCondition != nil && trafficPolicyCondition.Status == metav1.ConditionFalse {
		trafficPolicyReady = false
	}

	// Check if domain is ready
	domainReady := domainResult.IsReady

	// Overall ready status - all conditions must be true
	ready := endpointCreated && trafficPolicyReady && domainReady

	// Determine reason and message based on state
	var reason ConditionReadyReason
	var message string
	switch {
	case ready:
		reason = ReasonActive
		message = "AgentEndpoint is active and ready"
	case !domainReady:
		// Use the domain's Ready condition reason/message for better context
		if domainResult.ReadyReason != "" {
			reason = ConditionReadyReason(domainResult.ReadyReason)
			message = domainResult.ReadyMessage
		} else {
			reason = ReasonDomainNotReady
			message = "Domain is not ready"
		}
	case !endpointCreated:
		// If EndpointCreated condition exists and is False, use its reason/message
		if endpointCreatedCondition != nil && endpointCreatedCondition.Status == metav1.ConditionFalse {
			reason = ConditionReadyReason(endpointCreatedCondition.Reason)
			message = endpointCreatedCondition.Message
		} else {
			reason = ReasonPending
			message = "Waiting for endpoint creation"
		}
	case !trafficPolicyReady:
		// Use the traffic policy's condition reason/message
		reason = ConditionReadyReason(trafficPolicyCondition.Reason)
		message = trafficPolicyCondition.Message
	default:
		reason = ReasonUnknown
		message = "AgentEndpoint is not ready"
	}

	setReadyCondition(aep, ready, reason, message)
}
