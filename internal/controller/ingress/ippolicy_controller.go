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

package ingress

import (
	"context"
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"github.com/ngrok/ngrok-api-go/v7"
	ingressv1alpha1 "github.com/ngrok/ngrok-operator/api/ingress/v1alpha1"
	"github.com/ngrok/ngrok-operator/internal/controller"
	"github.com/ngrok/ngrok-operator/internal/ngrokapi"
)

const (
	IPPolicyRuleActionAllow = "allow"
	IPPolicyRuleActionDeny  = "deny"
)

// IPPolicyReconciler reconciles a IPPolicy object
type IPPolicyReconciler struct {
	client.Client

	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder

	IPPoliciesClient    ngrokapi.IPPoliciesClient
	IPPolicyRulesClient ngrokapi.IPPolicyRulesClient

	controller *controller.BaseController[*ingressv1alpha1.IPPolicy]
}

// SetupWithManager sets up the controller with the Manager.
func (r *IPPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.IPPoliciesClient == nil {
		return errors.New("IPPoliciesClient must be set")
	}
	if r.IPPolicyRulesClient == nil {
		return errors.New("IPPolicyRulesClient must be set")
	}

	r.controller = &controller.BaseController[*ingressv1alpha1.IPPolicy]{
		Kube:     r.Client,
		Log:      r.Log,
		Recorder: r.Recorder,

		StatusID: func(cr *ingressv1alpha1.IPPolicy) string { return cr.Status.ID },
		Create:   r.create,
		Update:   r.update,
		Delete:   r.delete,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&ingressv1alpha1.IPPolicy{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=ingress.k8s.ngrok.com,resources=ippolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ingress.k8s.ngrok.com,resources=ippolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ingress.k8s.ngrok.com,resources=ippolicies/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *IPPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return r.controller.Reconcile(ctx, req, new(ingressv1alpha1.IPPolicy))
}

func (r *IPPolicyReconciler) create(ctx context.Context, policy *ingressv1alpha1.IPPolicy) error {
	remotePolicy, err := r.IPPoliciesClient.Create(ctx, &ngrok.IPPolicyCreate{
		Description: policy.Spec.Description,
		Metadata:    policy.Spec.Metadata,
	})
	if err != nil {
		return err
	}
	policy.Status.ID = remotePolicy.ID
	if err := r.Status().Update(ctx, policy); err != nil {
		return err
	}

	return r.createOrUpdateIPPolicyRules(ctx, policy)
}

func (r *IPPolicyReconciler) update(ctx context.Context, policy *ingressv1alpha1.IPPolicy) error {
	remotePolicy, err := r.IPPoliciesClient.Get(ctx, policy.Status.ID)
	if err != nil {
		if ngrok.IsNotFound(err) {
			policy.Status.ID = ""
			return r.Status().Update(ctx, policy)
		}
		return err
	}

	if remotePolicy.Description != policy.Spec.Description ||
		remotePolicy.Metadata != policy.Spec.Metadata {
		r.Recorder.Event(policy, v1.EventTypeNormal, "Updating", fmt.Sprintf("Updating IPPolicy %s", policy.Name))
		_, err := r.IPPoliciesClient.Update(ctx, &ngrok.IPPolicyUpdate{
			ID:          policy.Status.ID,
			Description: ptr.To(policy.Spec.Description),
			Metadata:    ptr.To(policy.Spec.Metadata),
		})
		if err != nil {
			return err
		}
		r.Recorder.Event(policy, v1.EventTypeNormal, "Updated", fmt.Sprintf("Updated IPPolicy %s", policy.Name))
	}

	return r.createOrUpdateIPPolicyRules(ctx, policy)
}

func (r *IPPolicyReconciler) delete(ctx context.Context, policy *ingressv1alpha1.IPPolicy) error {
	err := r.IPPoliciesClient.Delete(ctx, policy.Status.ID)
	if err == nil || ngrok.IsNotFound(err) {
		policy.Status.ID = ""
	}
	return err
}

func (r *IPPolicyReconciler) createOrUpdateIPPolicyRules(ctx context.Context, policy *ingressv1alpha1.IPPolicy) error {
	log := ctrl.LoggerFrom(ctx)

	remoteRules, err := r.getRemotePolicyRules(ctx, policy.Status.ID)
	if err != nil {
		return err
	}
	iter := newIPPolicyDiff(policy.Status.ID, remoteRules, policy.Spec.Rules)

	for iter.Next() {
		for _, d := range iter.NeedsDelete() {
			log.V(3).Info("Deleting IP Policy Rule", "id", d.ID, "policy.id", policy.Status.ID, "cidr", d.CIDR, "action", d.Action)
			if err := r.IPPolicyRulesClient.Delete(ctx, d.ID); err != nil {
				return err
			}
			log.V(3).Info("Deleted IP Policy Rule", "id", d.ID)
		}

		for _, c := range iter.NeedsCreate() {
			log.V(3).Info("Creating IP Policy Rule", "policy.id", policy.Status.ID, "cidr", c.CIDR, "action", c.Action)
			rule, err := r.IPPolicyRulesClient.Create(ctx, c)
			if err != nil {
				return err
			}
			log.V(3).Info("Created IP Policy Rule", "id", rule.ID, "policy.id", policy.Status.ID, "cidr", rule.CIDR, "action", rule.Action)
		}

		for _, u := range iter.NeedsUpdate() {
			log.V(3).Info("Updating IP Policy Rule", "id", u.ID, "policy.id", policy.Status.ID, "cidr", u.CIDR, "metadata", u.Metadata, "description", u.Description)
			rule, err := r.IPPolicyRulesClient.Update(ctx, u)
			if err != nil {
				return err
			}
			log.V(3).Info("Updated IP Policy Rule", "id", rule.ID, "policy.id", policy.Status.ID)
		}
	}

	return nil
}

func (r *IPPolicyReconciler) getRemotePolicyRules(ctx context.Context, policyID string) ([]*ngrok.IPPolicyRule, error) {
	iter := r.IPPolicyRulesClient.List(&ngrok.Paging{})
	rules := make([]*ngrok.IPPolicyRule, 0)

	for iter.Next(ctx) {
		rule := iter.Item()
		// Filter to only rules that contain this policy ID
		if rule.IPPolicy.ID == policyID {
			rules = append(rules, rule)
		}
	}

	return rules, iter.Err()
}

// IPPolicyDiff represents the diff between the remote and spec rules for an IPPolicy.
// From the ngrok docs:
//
//	"IP Restrictions allow you to attach one or more IP policies to the route.
//	 If multiple IP policies are attached, a connection will be allowed only if
//	 its source IP matches at least one policy with an 'allow' action and
//	 does not match any policy with a 'deny' action."
//
// This provides an iterator of the rules that need to be created,updated, and deleted in order to update the remote securely.
type IPPolicyDiff struct {
	idx int

	policyID string

	remoteDeny  map[string]*ngrok.IPPolicyRule
	remoteAllow map[string]*ngrok.IPPolicyRule
	specDeny    map[string]ingressv1alpha1.IPPolicyRule
	specAllow   map[string]ingressv1alpha1.IPPolicyRule

	creates []*ngrok.IPPolicyRuleCreate
	deletes []*ngrok.IPPolicyRule
	updates []*ngrok.IPPolicyRuleUpdate
}

func newIPPolicyDiff(policyID string, remote []*ngrok.IPPolicyRule, spec []ingressv1alpha1.IPPolicyRule) *IPPolicyDiff {
	diff := &IPPolicyDiff{
		policyID:    policyID,
		remoteDeny:  make(map[string]*ngrok.IPPolicyRule),
		remoteAllow: make(map[string]*ngrok.IPPolicyRule),
		specDeny:    make(map[string]ingressv1alpha1.IPPolicyRule),
		specAllow:   make(map[string]ingressv1alpha1.IPPolicyRule),
	}

	// Group the remote rules by their CIDR
	for _, rule := range remote {
		if rule.Action == IPPolicyRuleActionDeny {
			diff.remoteDeny[rule.CIDR] = rule
		} else {
			diff.remoteAllow[rule.CIDR] = rule
		}
	}

	// Group the spec rules by their CIDR
	for _, rule := range spec {
		if rule.Action == IPPolicyRuleActionDeny {
			diff.specDeny[rule.CIDR] = rule
		} else {
			diff.specAllow[rule.CIDR] = rule
		}
	}

	return diff
}

func (d *IPPolicyDiff) Next() bool {
	defer func() { d.idx++ }()

	// Reset the diff
	d.creates = make([]*ngrok.IPPolicyRuleCreate, 0)
	d.deletes = make([]*ngrok.IPPolicyRule, 0)
	d.updates = make([]*ngrok.IPPolicyRuleUpdate, 0)

	switch d.idx {
	case 0: // Create all new deny rules that don't exist in the remote with a matching CIDR.
		for cidr, rule := range d.specDeny {
			if !d.existsInRemote(cidr) {
				d.creates = append(d.creates, d.createRule(rule))
			}
		}
		return true
	case 1: // Delete any allow rules with matching CIDRs that will be changing to deny rules. Then create the deny rules.
		for cidr, rule := range d.specDeny {
			if _, ok := d.remoteAllow[cidr]; ok {
				d.deletes = append(d.deletes, d.remoteAllow[cidr])
				d.creates = append(d.creates, d.createRule(rule))
			}
		}
		return true
	case 2: // Delete any deny rules with matching CIDRs that will be changing to allow rules. Then create the allow rules.
		for cidr, rule := range d.specAllow {
			if _, ok := d.remoteDeny[cidr]; ok {
				d.deletes = append(d.deletes, d.remoteDeny[cidr])
				d.creates = append(d.creates, d.createRule(rule))
			}
		}
		return true
	case 3: // Create all new allow rules that don't exist in the remote with a matching CIDR.
		for cidr, rule := range d.specAllow {
			if !d.existsInRemote(cidr) {
				d.creates = append(d.creates, d.createRule(rule))
			}
		}
		return true
	case 4: // Delete any remaining rules that are not in the spec.
		for cidr, rule := range d.remoteAllow {
			if !d.existsInSpec(cidr) {
				d.deletes = append(d.deletes, rule)
			}
		}
		for cidr, rule := range d.remoteDeny {
			if !d.existsInSpec(cidr) {
				d.deletes = append(d.deletes, rule)
			}
		}
		return true
	case 5: // Update any rules that exist in the spec and remote but have only different metadata/description.
		for cidr, rule := range d.specAllow {
			if remoteRule, ok := d.remoteAllow[cidr]; ok {
				d.addUpdateIfNeeded(rule, remoteRule)
			}
		}
		for cidr, rule := range d.specDeny {
			if remoteRule, ok := d.remoteDeny[cidr]; ok {
				d.addUpdateIfNeeded(rule, remoteRule)
			}
		}
		return true
	default:
	}

	return false
}

func (d *IPPolicyDiff) NeedsCreate() []*ngrok.IPPolicyRuleCreate {
	return d.creates
}

func (d *IPPolicyDiff) NeedsDelete() []*ngrok.IPPolicyRule {
	return d.deletes
}

func (d *IPPolicyDiff) NeedsUpdate() []*ngrok.IPPolicyRuleUpdate {
	return d.updates
}

// existsInSpec returns true if the CIDR exists in the spec for either an allow or deny rule.
func (d *IPPolicyDiff) existsInSpec(cidr string) bool {
	_, okDeny := d.specDeny[cidr]
	_, okAllow := d.specAllow[cidr]
	return okDeny || okAllow
}

// existsInRemote returns true if the CIDR exists in the remote for either an allow or deny rule.
func (d *IPPolicyDiff) existsInRemote(cidr string) bool {
	_, okDeny := d.remoteDeny[cidr]
	_, okAllow := d.remoteAllow[cidr]
	return okDeny || okAllow
}

func (d *IPPolicyDiff) createRule(rule ingressv1alpha1.IPPolicyRule) *ngrok.IPPolicyRuleCreate {
	return &ngrok.IPPolicyRuleCreate{
		IPPolicyID:  d.policyID,
		CIDR:        rule.CIDR,
		Action:      ptr.To(rule.Action),
		Metadata:    rule.Metadata,
		Description: rule.Description,
	}
}

func (d *IPPolicyDiff) addUpdateIfNeeded(rule ingressv1alpha1.IPPolicyRule, remoteRule *ngrok.IPPolicyRule) {
	updatedNeeded := rule.CIDR == remoteRule.CIDR &&
		rule.Action == remoteRule.Action &&
		(rule.Metadata != remoteRule.Metadata || rule.Description != remoteRule.Description)
	if !updatedNeeded {
		return
	}

	d.updates = append(d.updates, &ngrok.IPPolicyRuleUpdate{
		ID:          remoteRule.ID,
		Metadata:    ptr.To(rule.Metadata),
		Description: ptr.To(rule.Description),
		CIDR:        ptr.To(rule.CIDR),
	})
}
