/*
Copyright 2026.

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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	computev1 "github.com/r4rajat/cloud-resource-controller/api/v1"
)

// EC2InstanceReconciler reconciles a EC2Instance object
type EC2InstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=compute.r4rajat.com,resources=ec2instances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=compute.r4rajat.com,resources=ec2instances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=compute.r4rajat.com,resources=ec2instances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EC2Instance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/reconcile
func (r *EC2InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)

	logger.Info("------------Reconcilation Started------------", "name", req.Name, "namespace", req.Namespace)

	// Fetch the EC2Instance object
	ec2InstanceObject := computev1.EC2Instance{}
	if err := r.Get(ctx, req.NamespacedName, &ec2InstanceObject); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("EC2Instance resource not found. Ignoring since object must be deleted")
			// Kubernetes will not retry -> done. Waits for next event
			return ctrl.Result{}, nil
		}
		// Kubernetes wil retry with exponential backoff -> done. Waits for next event
		return ctrl.Result{}, err
	}

	logger.Info("EC2Instance resource found. Reconciling...")

	// In order to maintain idempotency, we check if the Object we get has Status field set
	// Only set when it has been processes already. If exists, then do nothing
	if ec2InstanceObject.Status.InstanceID != "" {
		logger.Info("Requested object already processed. Not creating a new instance", "instanceID", ec2InstanceObject.Status.InstanceID)
		return ctrl.Result{}, nil
	}

	logger.Info("-------------Creating New EC2Instance -------------", "name", ec2InstanceObject.Name)

	logger.Info("-------------Adding Finalizers--------------")

	// Finalizers are used to perform cleanup before the object is deleted.
	// Here, we add a finalizer to the EC2Instance object to ensure that any necessary cleanup is performed
	// before the object is removed from the cluster. This is important for managing external resources,
	// such as AWS EC2 instances, that may need to be terminated or cleaned up when the Kubernetes resource is deleted.

	if err := r.SetupFinalizer(ctx, &ec2InstanceObject); err != nil {
		logger.Error(err, "Failed to setup finalizer for EC2Instance")
		// Kubernetes will retry with exponential backoff -> done. Waits for next event
		return ctrl.Result{
			Requeue: true,
		}, err
	}

	logger.Info("-------------Finalizers Added -------------", "note", "This Update will trigger the Reconcile() function again, but current reconcilation continues.")
	logger.Info("-------------Creating EC2Instance on AWS in current Reconcile-------------", "name", ec2InstanceObject.Name)

	createdInstanceInfo, err := createEC2Instance(ctx, &ec2InstanceObject)
	if err != nil {
		logger.Error(err, "Failed to create EC2 instance on AWS")
		// Kubernetes will retry with exponential backoff -> done. Waits for next event
		return ctrl.Result{
			Requeue: true,
		}, err
	}

	logger.Info("EC2 Instance Created", "instanceID", createdInstanceInfo.InstanceID, "state", createdInstanceInfo.State)

	logger.Info("About to Update Status - This will trigger the Reconcile() function again, but current reconcilation continues.")

	if err = r.SetStatus(ctx, &ec2InstanceObject, createdInstanceInfo); err != nil {
		logger.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	logger.Info("-------------Reconcilation Done -------------")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EC2InstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&computev1.EC2Instance{}).
		Named("ec2instance").
		Complete(r)
}

func (r *EC2InstanceReconciler) SetupFinalizer(ctx context.Context, ec2Instance *computev1.EC2Instance) error {
	logger := logf.FromContext(ctx)

	// Check if the finalizer is already present
	if !containsString(ec2Instance.Finalizers, "ec2instance.compute.r4rajat.com") {
		logger.Info("Adding finalizer to EC2Instance", "name", ec2Instance.Name)
		ec2Instance.Finalizers = append(ec2Instance.Finalizers, "ec2instance.compute.r4rajat.com")
		if err := r.Update(ctx, ec2Instance); err != nil {
			logger.Error(err, "Failed to add finalizer to EC2Instance")
			return err
		}
		logger.Info("Finalizer added to EC2Instance", "name", ec2Instance.Name)
	}
	return nil
}

func (r *EC2InstanceReconciler) SetStatus(ctx context.Context, ec2InstanceObject *computev1.EC2Instance, createdInstanceInfo *computev1.CreatedEC2InstanceInfo) error {
	ec2InstanceObject.Status.InstanceID = createdInstanceInfo.InstanceID
	ec2InstanceObject.Status.State = createdInstanceInfo.State
	ec2InstanceObject.Status.PublicIP = createdInstanceInfo.PublicIP
	ec2InstanceObject.Status.PrivateIP = createdInstanceInfo.PrivateIP
	ec2InstanceObject.Status.PublicDNS = createdInstanceInfo.PublicDNS
	ec2InstanceObject.Status.PrivateDNS = createdInstanceInfo.PrivateDNS

	if err := r.Status().Update(ctx, ec2InstanceObject); err != nil {
		return err
	}

	return nil
}

func containsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}
