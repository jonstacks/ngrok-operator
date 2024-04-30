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
package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	ingressv1alpha1 "github.com/ngrok/kubernetes-ingress-controller/api/ingress/v1alpha1"
	"github.com/ngrok/kubernetes-ingress-controller/internal/controller/controllers"
	"github.com/ngrok/kubernetes-ingress-controller/internal/store"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	Recorder  record.EventRecorder
	Namespace string
	Driver    *store.Driver
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Owns(&ingressv1alpha1.Tunnel{}).
		Owns(&ingressv1alpha1.TCPEdge{}).
		Complete(r)
}

// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;update
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;list;watch;patch;update
// +kubebuilder:rbac:groups=ingress.k8s.ngrok.com,resources=ngrokmodulesets,verbs=get;list;watch
// +kubebuilder:rbac:groups=ingress.k8s.ngrok.com,resources=tunnels,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=ingress.k8s.ngrok.com,resources=tcpedges,verbs=get;list;watch;create;update;delete

// This reconcile function is called by the controller-runtime manager.
// It is invoked whenever there is an event that occurs for a resource
// being watched (in our case, service objects). If you tail the controller
// logs and delete, update, edit service objects, you see the events come in.
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("service", req.NamespacedName)
	ctx = ctrl.LoggerInto(ctx, log)

	svc := &corev1.Service{}
	if err := r.Client.Get(ctx, req.NamespacedName, svc); err != nil {
		log.Error(err, "unable to fetch service")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the service owns a tunnel or TCP Edge
	tunnels := ingressv1alpha1.TunnelList{}
	if err := r.Client.List(ctx, &tunnels, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "Failed to list tunnels")
		return ctrl.Result{}, err
	}

	edges := &ingressv1alpha1.TCPEdgeList{}
	if err := r.Client.List(ctx, edges, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "Failed to list TCP edges")
		return ctrl.Result{}, err
	}

	if !controllers.IsUpsert(svc) {
		log.Info("Service is being deleted, checking if it is being used by a tunnel or TCP edge")
		if len(tunnels.Items) > 0 || len(edges.Items) > 0 {
			log.Info("service is being used by a tunnel or TCP edge, skipping deletion")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}

		log.Info("Removing and syncing finalizer")
		if controllers.HasFinalizer(svc) {
			if err := controllers.RemoveAndSyncFinalizer(ctx, r.Client, svc); err != nil {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		log.Info("Service is not of type LoadBalancer, performing any cleanup and then skipping")
		// We need to check if the service is being changed from a LoadBalancer to something else.
		// If it is, we need to clean up any tunnels or TCP edges that are using it
		if len(tunnels.Items) > 0 {
			log.Info("Service is being changed from LoadBalancer to something else, deleting tunnels")
			for _, tunnel := range tunnels.Items {
				if metav1.IsControlledBy(&tunnel, svc) {
					if err := r.Client.Delete(ctx, &tunnel); err != nil {
						log.Error(err, "Failed to delete tunnel", "tunnel", tunnel)
						return ctrl.Result{}, err
					}
				}
			}
		}
		if len(edges.Items) > 0 {
			log.Info("Service is being changed from LoadBalancer to something else, deleting TCP edges")
			for _, edge := range edges.Items {
				if metav1.IsControlledBy(&edge, svc) {
					if err := r.Client.Delete(ctx, &edge); err != nil {
						log.Error(err, "Failed to delete TCP edge", "edge", edge)
						return ctrl.Result{}, err
					}
				}
			}
		}
		// Once we clean up the tunnels and TCP edges, we can remove the finalizer if it exists. We don't
		// care about registering a finalizer since we only care about load balancer services
		if err := controllers.RemoveAndSyncFinalizer(ctx, r.Client, svc); err != nil {
			log.Error(err, "Failed to remove finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if len(svc.Spec.Ports) < 1 {
		log.Info("Service has no ports, skipping")
		return ctrl.Result{}, nil
	}

	log.Info("Registering and syncing finalizers")
	if err := controllers.RegisterAndSyncFinalizer(ctx, r.Client, svc); err != nil {
		log.Error(err, "Failed to register finalizer")
		return ctrl.Result{}, err
	}

	// There is some impedance mismatch between the k8s services and TCP/TLS tunnels, so we
	// can only support services with a single port for now.
	port := svc.Spec.Ports[0].Port

	backendLabels := map[string]string{
		"k8s.ngrok.com/namespace":   svc.Namespace,
		"k8s.ngrok.com/service":     svc.Name,
		"k8s.ngrok.com/service-uid": string(svc.UID),
		"k8s.ngrok.com/port":        string(port),
	}

	tunnel := &ingressv1alpha1.Tunnel{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: svc.Name + "-",
			Namespace:    svc.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(svc, corev1.SchemeGroupVersion.WithKind("Service")),
			},
		},
		Spec: ingressv1alpha1.TunnelSpec{
			ForwardsTo: fmt.Sprintf("%s.%s.%s:%d", svc.Name, svc.Namespace, "svc.cluster.local", port),
			Labels:     backendLabels,
		},
	}

	edge := &ingressv1alpha1.TCPEdge{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: svc.Name + "-",
			Namespace:    svc.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(svc, corev1.SchemeGroupVersion.WithKind("Service")),
			},
		},
		Spec: ingressv1alpha1.TCPEdgeSpec{
			Backend: ingressv1alpha1.TunnelGroupBackend{
				Labels: backendLabels,
			},
		},
	}

	// Create the tunnel and TCP edge if they don't exist, otherwise update them
	ownedTunnels := []ingressv1alpha1.Tunnel{}
	for _, t := range tunnels.Items {
		t := t
		if metav1.IsControlledBy(&t, svc) {
			ownedTunnels = append(ownedTunnels, t)
		}
	}

	ownedEdges := []ingressv1alpha1.TCPEdge{}
	for _, e := range edges.Items {
		e := e
		if metav1.IsControlledBy(&e, svc) {
			ownedEdges = append(ownedEdges, e)
		}
	}

	if len(ownedTunnels) == 1 {
		if !reflect.DeepEqual(ownedTunnels[0].Spec, tunnel.Spec) {
			log.Info("Updating tunnel", "tunnel", ownedTunnels[0])
			ownedTunnels[0].Spec = tunnel.Spec
			if err := r.Client.Update(ctx, &ownedTunnels[0]); err != nil {
				log.Error(err, "Failed to update tunnel", "tunnel", ownedTunnels[0])
				return ctrl.Result{}, err
			}
		}
	} else {
		if len(ownedTunnels) > 1 {
			log.Info("Found multiple tunnels owned by service, deleting extra tunnels")
			for _, t := range ownedTunnels[1:] {
				if err := r.Client.Delete(ctx, &t); err != nil {
					log.Error(err, "Failed to delete tunnel", "tunnel", t)
					return ctrl.Result{}, err
				}
			}
		}

		log.Info("Creating tunnel", "tunnel", tunnel)
		if err := r.Client.Create(ctx, tunnel); err != nil {
			log.Error(err, "Failed to create tunnel", "tunnel", tunnel)
			return ctrl.Result{}, err
		}
	}

	if len(ownedEdges) == 1 {
		if !reflect.DeepEqual(ownedEdges[0].Spec, edge.Spec) {
			log.Info("Updating TCP edge", "edge", ownedEdges[0])
			ownedEdges[0].Spec = edge.Spec
			if err := r.Client.Update(ctx, &ownedEdges[0]); err != nil {
				log.Error(err, "Failed to update TCP edge", "edge", ownedEdges[0])
				return ctrl.Result{}, err
			}
			edge = &ownedEdges[0]
		}
	} else {
		if len(ownedEdges) > 1 {
			log.Info("Found multiple TCP edges owned by service, deleting extra edges")
			for _, e := range ownedEdges[1:] {
				if err := r.Client.Delete(ctx, &e); err != nil {
					log.Error(err, "Failed to delete TCP edge", "edge", e)
					return ctrl.Result{}, err
				}
			}
		}

		log.Info("Creating TCP edge", "edge", edge)
		if err := r.Client.Create(ctx, edge); err != nil {
			log.Error(err, "Failed to create TCP edge", "edge", edge)
			return ctrl.Result{}, err
		}
	}

	// Update the status of the service with the hostport of the TCP edge
	if len(edge.Status.Hostports) == 0 {
		log.Info("TCP edge has no hostports, skipping status update")
		return ctrl.Result{}, nil
	}

	hostport := edge.Status.Hostports[0]

	pieces := strings.SplitN(hostport, ":", 2)
	if len(pieces) != 2 {
		err := fmt.Errorf("invalid hostport")
		log.Error(err, "error parsing hostport into hostname and port", "hostport", hostport)
		return ctrl.Result{}, err
	}

	ngrokPort, err := strconv.ParseInt(pieces[1], 10, 32)
	if err != nil {
		log.Error(err, "error parsing port", "port", pieces[1])
		return ctrl.Result{}, err
	}

	// Update the service status
	svc.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
		{
			Hostname: pieces[0],
			Ports: []corev1.PortStatus{
				{
					Port:     int32(ngrokPort),
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	if err := r.Client.Status().Update(ctx, svc); err != nil {
		log.Error(err, "Failed to update service status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
