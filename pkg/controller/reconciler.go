// Copyright 2020-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"

	"github.com/atomix/kubernetes-controller/pkg/apis/cloud/v1beta2"
	"github.com/atomix/kubernetes-controller/pkg/controller/v1beta2/storage"
	"github.com/atomix/redis-storage-controller/pkg/apis/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var log = logf.Log.WithName("redis_controller")

// Add creates a new Partition ManagementGroup and adds it to the Manager. The Manager will set fields on the ManagementGroup
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	log.Info("Add manager")
	reconciler := &Reconciler{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
	}
	gvk := schema.GroupVersionKind{
		Group:   v1beta1.RedisStorageClassGroup,
		Version: v1beta1.RedisStorageClassVersion,
		Kind:    v1beta1.RedisStorageClassKind,
	}
	return storage.AddClusterReconciler(mgr, reconciler, gvk)
}

var _ reconcile.Reconciler = &Reconciler{}

// Reconciler reconciles a Cluster object
type Reconciler struct {
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Info("Reconcile Redis Cluster")
	cluster := &v1beta2.Cluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, cluster)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{Requeue: true}, err
	}

	storage := &v1beta1.RedisStorageClass{}
	name := types.NamespacedName{
		Namespace: cluster.Spec.Storage.Namespace,
		Name:      cluster.Spec.Storage.Name,
	}
	err = r.client.Get(context.TODO(), name, storage)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{Requeue: true}, err
	}

	err = r.reconcileBackendConfigMap(cluster, storage)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return reconcile.Result{}, err
		}

	}

	err = r.reconcileBackendHeadlessService(cluster, storage)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return reconcile.Result{}, err
		}
	}

	err = r.reconcileBackendStatefulSet(cluster, storage)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.reconcileBackendService(cluster, storage)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.reconcileProxyDeployment(cluster, storage)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.reconcileProxyService(cluster, storage)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
