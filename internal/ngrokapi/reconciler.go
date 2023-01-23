package ngrokapi

import (
	"context"
	"fmt"

	"github.com/ngrok/ngrok-api-go/v5"
)

const (
	ReconcileActionDelete         = iota
	ReconcileActionCreateOrUpdate = iota
)

type Logger interface {
	Info(string, ...interface{})
	Error(error, string, ...interface{})
}

type IsSearcReconcilable interface {
	// SearchForMatchingObject searches the remote API for an object that matches
	// the current object. The logic to determine match is up to the implementor.
	SearchForMatchingObject(context.Context) (bool, error)
}

// Reconcilable is something that is able to be reconciled with the ngrok API.
type Reconcilable interface {
	// APIType returns the type of the object in the ngrok API to be reconcilled.
	// Ex: HTTPSEdge
	APIType() string

	// Logger returns the logger to be used for logging.
	// If the Reconcilable does not have a logger, it should return nil.
	Logger() Logger

	// ID returns the ID of the object if it exists. If the Reconcilable does not have an
	// ID, it should return nil.
	ID() *string

	// Get should retrieve the object from the ngrok API.
	Get(context.Context) error

	// Create creates a new object in the ngrok API.
	Create(context.Context) error

	// Update updates the object in the ngrok API.
	Update(context.Context) error

	// Delete deletes the object from the ngrok API.
	Delete(context.Context) error
}

// ReconcilableWithSearch is a Reconcilable that can be adopted by matching it to an existing
// object in the remote API. Different API objects have different ways of matching, so the
// logic to determine match is up to the implementor.
type ReconcilableWithSearch interface {
	Reconcilable
	IsSearcReconcilable
}

// In order to DRY up the common logic of reconciling API object, we use a reconciler that is
// conceptually similar to a controlelr. The reconciler is responsible for determining if an
// object needs to be created, updated, or deleted. It has the common logic for what to do
// when an object is missing, or if an object can be adopted by matching it to an existing
// object in the remote API.
type Reconciler struct{}

func (r *Reconciler) Reconcile(ctx context.Context, obj Reconcilable, action int) error {
	switch action {
	case ReconcileActionDelete:
		return r.deleteObjectIfExists(ctx, obj)
	case ReconcileActionCreateOrUpdate:
		return r.createOrUpdateObject(ctx, obj)
	}

	return fmt.Errorf("Unsuported action: %d", action)
}

func (r *Reconciler) deleteObjectIfExists(ctx context.Context, obj Reconcilable) error {
	logger := newWrappedLogger(obj)

	// delete the object if it exists
	if obj.ID() == nil {
		logger.Info("Skipping delete because object has no ID")
		return nil
	}

	logger.Info("Deleting object", "apiType", obj.APIType(), "id", *obj.ID())
	return obj.Delete(ctx)
}

func (r *Reconciler) createOrUpdateObject(ctx context.Context, obj Reconcilable) error {
	logger := newWrappedLogger(obj)

	if obj.ID() != nil {
		logger.Info("Object exists, checking if it still exists in the remote API...")
		err := obj.Get(ctx)
		if err == nil {
			logger.Info("Found object by ID, updating...")
			return obj.Update(ctx)
		}

		// Some other error besides not found
		if !ngrok.IsNotFound(err) {
			return err
		}

		// Not found error, so we need to find or create the object
		logger.Info("Object with ID is missing in the remote API")
		return r.findOrCreate(ctx, obj)
	}

	// create or update the object
	logger.Info("Object does not exist, finding or creating...")
	return r.findOrCreate(ctx, obj)
}

func (r *Reconciler) findOrCreate(ctx context.Context, obj Reconcilable) error {
	logger := newWrappedLogger(obj)

	// If the reconcilable we received also implements ReconcliableWithSearch, then
	// we can search for an existing object that matches.
	if v, ok := obj.(ReconcilableWithSearch); ok {
		logger.Info("Searching for matching object...")

		found, err := v.SearchForMatchingObject(ctx)
		if err != nil {
			logger.Error(err, "Failed to search for matching object")
			return err
		}

		if found {
			logger.Info("Found matching object")
			return nil
		}

		logger.Info("No match of existing object found")
	}

	logger.Info("Creating new object...")
	err := obj.Create(ctx)
	if err != nil {
		logger.Error(err, "Failed to create object")
	}
	return err
}

// Wrapped Logger wraps the object we are reconciling to determine if the logger is nil
// and also to add extra context like the object ID and API type.
type wrappedLogger struct {
	obj Reconcilable
}

func newWrappedLogger(obj Reconcilable) wrappedLogger {
	return wrappedLogger{
		obj: obj,
	}
}

func (w *wrappedLogger) Info(msg string, keysAndValues ...interface{}) {
	if w.obj.Logger() != nil {
		keysAndValues = append(keysAndValues, "apiType", w.obj.APIType(), "id", w.obj.ID())
		w.obj.Logger().Info(msg, keysAndValues...)
	}
}

func (w *wrappedLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	if w.obj.Logger() != nil {
		keysAndValues = append(keysAndValues, "apiType", w.obj.APIType(), "id", w.obj.ID())
		w.obj.Logger().Error(err, msg, keysAndValues...)
	}
}
