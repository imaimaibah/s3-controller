/*
Copyright 2022.

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

package controllers

import (
	//"fmt"
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	s3v1alpha1 "github.com/imaimaibah/s3-controller/api/v1alpha1"

	"github.com/go-logr/logr"
	"k8s.io/client-go/tools/record"

	s3 "github.com/imaimaibah/s3-controller/pkg/s3"
)

// BucketReconciler reconciles a Bucket object
type BucketReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=s3.sedex.io,resources=buckets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=s3.sedex.io,resources=buckets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=s3.sedex.io,resources=buckets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Bucket object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *BucketReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	log := r.Log.WithValues("bucket", req.NamespacedName)

	var bucket s3v1alpha1.Bucket
	log.Info("fetching Bucket Resource")
	if err := r.Get(ctx, req.NamespacedName, &bucket); err != nil {
		log.Error(err, "unable to fetch Bucket")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// name of our custom finalizer
	myFinalizerName := "bucket.sedex.io/finalizer"
	// examine DeletionTimestamp to determine if object is under deletion
	if bucket.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(&bucket, myFinalizerName) {
			controllerutil.AddFinalizer(&bucket, myFinalizerName)
			if err := r.Update(ctx, &bucket); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(&bucket, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(bucket.ObjectMeta.Name); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&bucket, myFinalizerName)
			if err := r.Update(ctx, &bucket); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	//fmt.Printf("%+v\n", bucket)
	log.Info(bucket.ObjectMeta.Name)
	log.Info(bucket.Spec.Versioning)
	log.Info(bucket.Spec.Encrypt)

	objectName := bucket.ObjectMeta.Name
	versioning := bucket.Spec.Versioning
	encryption := bucket.Spec.Encrypt
	err := s3.Create(objectName)
	if err != nil {
		return ctrl.Result{}, nil
	}
	err = s3.Update(objectName, versioning, encryption)
	if err != nil {
		return ctrl.Result{}, nil
	}
	//UpdateStatus()

	//updateBucketStatus()
	return ctrl.Result{}, nil
}

//func (r *BucketReconciler) deleteExternalResources(obj s3v1alpha1.Bucket) error {
func (r *BucketReconciler) deleteExternalResources(objectName string) error {
	//objectName := obj.ObjectMeta.Name
	if err := s3.Delete(objectName); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BucketReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&s3v1alpha1.Bucket{}).
		Complete(r)
}
