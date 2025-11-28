/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nodelabelsv1 "github.com/cloud-for-you/node-label-operator/api/v1"
	"github.com/cloud-for-you/node-label-operator/pkg"
	"github.com/go-logr/logr"
)

const (
	labelsFinalizer = "node-labels.cfy.cz/finalizer"
)

// LabelsReconciler reconciles a Labels object
type LabelsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

//+kubebuilder:rbac:groups=node-labels.cfy.cz,resources=labels,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch
//+kubebuilder:rbac:groups=node-labels.cfy.cz,resources=labels/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=node-labels.cfy.cz,resources=labels/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Labels object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *LabelsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here
	log := log.FromContext(ctx)
	log.Info("Verify if a CRD of Labels exists")

	// Fetch the Labels instance
	labels := &nodelabelsv1.Labels{}
	err := r.Get(ctx, req.NamespacedName, labels)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Labels resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Labels")
		return ctrl.Result{}, err
	}

	// Check if the Labels instance is Marked to be deleted
	isLabelsMarkedToBeDeleted := labels.GetDeletionTimestamp() != nil
	if isLabelsMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(labels, labelsFinalizer) {
			// Run finalization logic
			if err := r.finalizeLabels(labels); err != nil {
				return ctrl.Result{}, err
			}

			// Remove finalizer
			controllerutil.RemoveFinalizer(labels, labelsFinalizer)
			err := r.Update(ctx, labels)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// START
	// Iterujeme pres vsechny nody

	// ziskame vsechny labely, ktere budeme nastavovat
	allLabels := &nodelabelsv1.LabelsList{}
	if err = r.List(context.TODO(), allLabels, &client.ListOptions{}); err != nil {
		log.Error(err, "Failed to list Labels")
		return ctrl.Result{}, err
	}

	// ziskame seznam vsech nodu
	nodes := &v1.NodeList{}
	if err = r.List(context.TODO(), nodes, &client.ListOptions{}); err != nil {
		log.Error(err, "Failed to list Nodes")
		return ctrl.Result{}, err
	}

	// Nastavime labely
	for _, nodeOrig := range nodes.Items {
		log.Info("Processing node", "nodeName", nodeOrig.Name)

		node := nodeOrig.DeepCopy()
		nodeModified := false
		nodeModified = pkg.AddLabels(node, *labels, log) || nodeModified

		if nodeModified {
			log.Info("patching node")
			baseToPatch := client.MergeFrom(&nodeOrig)
			if err := r.Patch(context.TODO(), node, baseToPatch); err != nil {
				log.Error(err, "Failed to patch Node")
				return ctrl.Result{}, err
			}
		}

	}

	// END

	// Add finalizer for all CR
	if !controllerutil.ContainsFinalizer(labels, labelsFinalizer) {
		controllerutil.AddFinalizer(labels, labelsFinalizer)
		err = r.Update(ctx, labels)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Minute * 2}, nil
}

func (r *LabelsReconciler) finalizeLabels(m *nodelabelsv1.Labels) error {
	log.Log.Info("Successfully finalize labels")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LabelsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nodelabelsv1.Labels{}).
		Owns(&v1.Node{}).
		Complete(r)
}
