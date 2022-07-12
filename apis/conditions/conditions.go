package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reasons that give more context to conditions
const (
	ReasonAvailable          = "Available"
	ReasonProvisioningFailed = "ProvisioningFailed"
)

const (
	// TypeReady indicates that a resource is ready for use.
	TypeReady = "Ready"
	// TypeFailed indicates that a resource has failed the provisioning.
	TypeFailed = "Failed"
)

// Ready creates a condition with TypeReady, ReasonAvailable and empty message.
func Ready() metav1.Condition {
	return metav1.Condition{
		Type:               TypeReady,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonAvailable,
	}
}

// NotReady creates a condition with TypeReady, ReasonAvailable and empty message.
func NotReady() metav1.Condition {
	return metav1.Condition{
		Type:               TypeReady,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonAvailable,
	}
}

// Failed creates a condition with TypeFailed, ReasonProvisioningFailed and the error message.
func Failed(err error) metav1.Condition {
	return metav1.Condition{
		Type:               TypeFailed,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             ReasonProvisioningFailed,
		Message:            err.Error(),
	}
}

// ObjectWithConditions represents an object that has status conditions.
type ObjectWithConditions interface {
	client.Object
	// GetConditions returns the currently active conditions.
	GetConditions() []metav1.Condition
	// SetConditions sets the active conditions to the given conditions.
	SetConditions(v []metav1.Condition)
}
