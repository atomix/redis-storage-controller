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
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/atomix/kubernetes-controller/pkg/apis/cloud/v1beta2"
	"github.com/atomix/redis-storage-controller/pkg/apis/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// GetProxyDeploymentName returns the StatefulSet name for the given partition
func GetProxyDeploymentName(cluster *v1beta2.Cluster) string {
	return getClusterResourceName(cluster, getProxyName(cluster))
}

// getProxyName returns the name of the given cluster's proxy
func getProxyName(cluster *v1beta2.Cluster) string {
	return fmt.Sprintf("%s-proxy", cluster.Name)
}

func (r *Reconciler) reconcileProxyDeployment(cluster *v1beta2.Cluster, storage *v1beta1.RedisStorageClass) error {
	log.Info("Reconcile redis proxy deployment")
	dep := &appsv1.Deployment{}
	name := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      GetProxyDeploymentName(cluster),
	}
	err := r.client.Get(context.TODO(), name, dep)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addProxyDeployment(cluster, storage)
	}
	return err
}

func (r *Reconciler) addProxyDeployment(cluster *v1beta2.Cluster, storage *v1beta1.RedisStorageClass) error {
	log.Info("Creating Deployment", "Name", cluster.Name, "Namespace", cluster.Namespace)
	var replicas int32 = 1
	var env []corev1.EnvVar
	env = append(env, corev1.EnvVar{
		Name: "NODE_ID",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})

	volumes := []corev1.Volume{
		newConfigVolume(cluster.Name),
	}

	image := storage.Spec.Proxy.Image
	if image == "" {
		image = "atomix/redis-storage-node:latest"
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cluster.Namespace,
			Name:      GetProxyDeploymentName(cluster),
			Labels:    cluster.Labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: cluster.Labels,
			},
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetProxyDeploymentName(cluster),
					Namespace: cluster.Namespace,
					Labels:    cluster.Labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            GetProxyDeploymentName(cluster),
							Image:           image,
							ImagePullPolicy: storage.Spec.Proxy.ImagePullPolicy,
							Env:             env,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(cluster, dep, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), dep)
}
