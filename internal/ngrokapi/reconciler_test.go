package ngrokapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/ngrok/ngrok-api-go/v5"
	"github.com/stretchr/testify/assert"
)

var testReconciler = &Reconciler{}

func newNgrokNotFoundError() *ngrok.Error {
	return &ngrok.Error{
		StatusCode: http.StatusNotFound,
	}
}

type testLogger struct {
	t *testing.T
}

func (l *testLogger) Info(msg string, keysAndValues ...interface{}) {
	l.t.Logf(msg, keysAndValues...)
}

func (l *testLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.t.Logf(msg, keysAndValues...)
}

type testReconcilable struct {
	apiType string
	id      *string
	logger  Logger

	getCalled      bool
	getCalledError error

	createCalled bool
	createError  error

	updateCalled bool
	updateError  error

	deleteCalled bool
	deleteError  error

	searchForMatchCalled        bool
	searchForMatchShouldBeFound bool
	searchForMatchError         error
}

func newTestReconcilable(apiType string) *testReconcilable {
	return &testReconcilable{
		apiType: apiType,
	}
}

func (t *testReconcilable) APIType() string {
	return t.apiType
}

func (t *testReconcilable) ID() *string {
	return t.id
}

func (t *testReconcilable) WithID(id string) *testReconcilable {
	t.id = &id
	return t
}

func (t *testReconcilable) Logger() Logger {
	return t.logger
}

func (t *testReconcilable) WithTestLogger(tst *testing.T) *testReconcilable {
	t.logger = &testLogger{t: tst}
	return t
}

func (t *testReconcilable) Get(ctx context.Context) error {
	t.getCalled = true
	return t.getCalledError
}

func (t *testReconcilable) WithGetError(err error) *testReconcilable {
	t.getCalledError = err
	return t
}

func (t *testReconcilable) Create(ctx context.Context) error {
	t.createCalled = true
	return t.createError
}

func (t *testReconcilable) WithCreateError(err error) *testReconcilable {
	t.createError = err
	return t
}

func (t *testReconcilable) Update(ctx context.Context) error {
	t.updateCalled = true
	return t.updateError
}

func (t *testReconcilable) WithUpdateError(err error) *testReconcilable {
	t.updateError = err
	return t
}

func (t *testReconcilable) Delete(ctx context.Context) error {
	t.deleteCalled = true
	return t.deleteError
}

func (t *testReconcilable) WithDeleteError(err error) *testReconcilable {
	t.deleteError = err
	return t
}

func (t *testReconcilable) SearchForMatchingObject(ctx context.Context) (bool, error) {
	t.searchForMatchCalled = true
	return t.searchForMatchShouldBeFound, t.searchForMatchError
}

func (t *testReconcilable) WithSearchForMatchingObjectError(err error) *testReconcilable {
	t.searchForMatchError = err
	return t
}

func (t *testReconcilable) WithSearchForMatchingObjectShouldBeFound(shouldBeFound bool) *testReconcilable {
	t.searchForMatchShouldBeFound = shouldBeFound
	return t
}

var reconciler = &Reconciler{}

func assertOnlyCalled(t *testing.T, obj *testReconcilable, methods ...string) {
	shouldBeCalled := map[string]bool{
		"Get":                     false,
		"Create":                  false,
		"Update":                  false,
		"Delete":                  false,
		"SearchForMatchingObject": false,
	}

	for _, method := range methods {
		shouldBeCalled[method] = true
	}

	for method, shouldBeCalled := range shouldBeCalled {
		switch method {
		case "Get":
			assert.Equal(t, shouldBeCalled, obj.getCalled)
		case "Create":
			assert.Equal(t, shouldBeCalled, obj.createCalled)
		case "Update":
			assert.Equal(t, shouldBeCalled, obj.updateCalled)
		case "Delete":
			assert.Equal(t, shouldBeCalled, obj.deleteCalled)
		case "SearchForMatchingObject":
			assert.Equal(t, shouldBeCalled, obj.searchForMatchCalled)
		default:
			t.Fatalf("Unknown method: %s", method)
		}
	}
}

func TestBadAction(t *testing.T) {
	obj := newTestReconcilable("test").
		WithTestLogger(t)

	err := testReconciler.Reconcile(context.Background(), obj, 42)
	assert.Error(t, err)
	assertOnlyCalled(t, obj)
}

func TestDoesNotDeleteWhenDoesNotExist(t *testing.T) {
	obj := newTestReconcilable("test").
		WithTestLogger(t)

	err := testReconciler.Reconcile(context.Background(), obj, ReconcileActionDelete)
	assert.NoError(t, err)
	assertOnlyCalled(t, obj)
}

func TestDeleteWhenDoesExist(t *testing.T) {
	obj := newTestReconcilable("test").
		WithTestLogger(t).
		WithID("test-id")

	err := testReconciler.Reconcile(context.Background(), obj, ReconcileActionDelete)
	assert.NoError(t, err)
	assertOnlyCalled(t, obj, "Delete")
}

func TestCreateWhenDoesNotExistAndNoErrors(t *testing.T) {
	// Happy path, no errors
	obj := newTestReconcilable("test").
		WithTestLogger(t).
		WithSearchForMatchingObjectShouldBeFound(false)

	err := testReconciler.Reconcile(context.Background(), obj, ReconcileActionCreateOrUpdate)
	assert.NoError(t, err)
	assertOnlyCalled(t, obj, "SearchForMatchingObject", "Create")
}

func TestCreateWhenDoesNotExistAndASearchError(t *testing.T) {
	// Error when searching for matching object
	obj := newTestReconcilable("test").
		WithTestLogger(t).
		WithSearchForMatchingObjectError(assert.AnError)

	err := testReconciler.Reconcile(context.Background(), obj, ReconcileActionCreateOrUpdate)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	assertOnlyCalled(t, obj, "SearchForMatchingObject")
}

func TestCreateWhenDoesNotExistAndCreateError(t *testing.T) {
	// Error when creating object
	obj := newTestReconcilable("test").
		WithTestLogger(t).
		WithSearchForMatchingObjectShouldBeFound(false).
		WithCreateError(assert.AnError)

	err := testReconciler.Reconcile(context.Background(), obj, ReconcileActionCreateOrUpdate)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	assertOnlyCalled(t, obj, "SearchForMatchingObject", "Create")
}

func TestCreateWhenDoesNotExistAndMatchingObjectFound(t *testing.T) {
	// Matching object found
	obj := newTestReconcilable("test").
		WithTestLogger(t).
		WithSearchForMatchingObjectShouldBeFound(true)

	err := testReconciler.Reconcile(context.Background(), obj, ReconcileActionCreateOrUpdate)
	assert.NoError(t, err)
	assertOnlyCalled(t, obj, "SearchForMatchingObject")
}

func TestUpdateWhenExistsAndNoErrors(t *testing.T) {
	// Happy path, no errors
	obj := newTestReconcilable("test").
		WithTestLogger(t).
		WithID("test-id")

	err := testReconciler.Reconcile(context.Background(), obj, ReconcileActionCreateOrUpdate)
	assert.NoError(t, err)
	assertOnlyCalled(t, obj, "Get", "Update")
}

func TestUpdateWhenDoesNotExistCreatesInstead(t *testing.T) {
	obj := newTestReconcilable("test").
		WithID("test-id").
		WithTestLogger(t).
		WithGetError(newNgrokNotFoundError())

	err := testReconciler.Reconcile(context.Background(), obj, ReconcileActionCreateOrUpdate)
	assert.NoError(t, err)
	assertOnlyCalled(t, obj, "Get", "SearchForMatchingObject", "Create")
}

func TestUpdateWhenDoesNotExistAndSearchFails(t *testing.T) {
	obj := newTestReconcilable("test").
		WithID("test-id").
		WithTestLogger(t).
		WithGetError(newNgrokNotFoundError()).
		WithSearchForMatchingObjectError(assert.AnError)

	err := testReconciler.Reconcile(context.Background(), obj, ReconcileActionCreateOrUpdate)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	assertOnlyCalled(t, obj, "Get", "SearchForMatchingObject")
}

func TestUpdateWhenDoesNotExistAndCreateFails(t *testing.T) {
	obj := newTestReconcilable("test").
		WithID("test-id").
		WithTestLogger(t).
		WithGetError(newNgrokNotFoundError()).
		WithSearchForMatchingObjectShouldBeFound(false).
		WithCreateError(assert.AnError)

	err := testReconciler.Reconcile(context.Background(), obj, ReconcileActionCreateOrUpdate)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	assertOnlyCalled(t, obj, "Get", "SearchForMatchingObject", "Create")
}

func TestUpdateWhenGettingObjectInAPIFailsDueToOtherError(t *testing.T) {
	obj := newTestReconcilable("test").
		WithID("test-id").
		WithTestLogger(t).
		WithGetError(assert.AnError)

	err := testReconciler.Reconcile(context.Background(), obj, ReconcileActionCreateOrUpdate)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	assertOnlyCalled(t, obj, "Get")
}
