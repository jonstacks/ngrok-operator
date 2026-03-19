package controller

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/tools/record"
)

// eventRecorderAdapter wraps a record.EventRecorder to implement events.EventRecorder.
// This adapter bridges the old-style recorder with the new events API while
// preserving backward compatibility with the existing event broadcaster setup.
type eventRecorderAdapter struct {
	inner record.EventRecorder
}

// NewEventRecorderAdapter creates an events.EventRecorder from a record.EventRecorder.
func NewEventRecorderAdapter(recorder record.EventRecorder) events.EventRecorder {
	return &eventRecorderAdapter{inner: recorder}
}

func (a *eventRecorderAdapter) Eventf(regarding runtime.Object, related runtime.Object, eventtype, reason, action string, note string, args ...interface{}) {
	a.inner.Eventf(regarding, eventtype, reason, note, args...)
}
