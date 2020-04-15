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
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"k8s.io/apimachinery/pkg/types"

	api "github.com/atomix/api/proto/atomix/controller"
	"github.com/atomix/kubernetes-controller/pkg/apis/cloud/v1beta2"
	"github.com/atomix/redis-storage-controller/pkg/apis/v1beta1"
	storage "github.com/atomix/redis-storage-controller/pkg/apis/v1beta1"
	"github.com/gogo/protobuf/jsonpb"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *Reconciler) reconcileBackendConfigMap(cluster *v1beta2.Cluster, storage *v1beta1.RedisStorageClass) error {
	log.Info("Reconcile raft storage config map")
	cm := &corev1.ConfigMap{}
	name := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}
	err := r.client.Get(context.TODO(), name, cm)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addConfigMap(cluster, storage)
	}
	return err
}

func (r *Reconciler) reconcileBackendStatefulSet(cluster *v1beta2.Cluster, storage *v1beta1.RedisStorageClass) error {
	log.Info("Reconcile raft storage stateful set")
	statefulSet := &appsv1.StatefulSet{}
	name := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}
	err := r.client.Get(context.TODO(), name, statefulSet)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addStatefulSet(cluster, storage)
	}
	return err
}

func (r *Reconciler) reconcileBackendService(cluster *v1beta2.Cluster, storage *v1beta1.RedisStorageClass) error {
	log.Info("Reconcile raft storage service")
	service := &corev1.Service{}
	name := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}
	err := r.client.Get(context.TODO(), name, service)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addService(cluster, storage)
	}
	return err
}

func (r *Reconciler) reconcileBackendHeadlessService(cluster *v1beta2.Cluster, storage *v1beta1.RedisStorageClass) error {
	log.Info("Reconcile raft storage headless service")
	service := &corev1.Service{}
	name := types.NamespacedName{
		Namespace: cluster.Namespace,
		Name:      cluster.Name,
	}
	err := r.client.Get(context.TODO(), name, service)
	if err != nil && k8serrors.IsNotFound(err) {
		err = r.addHeadlessService(cluster, storage)
	}
	return err
}

// NewClusterConfigMap returns a new ConfigMap for initializing Atomix clusters
func NewClusterConfigMap(cluster *v1beta2.Cluster, storage *storage.RedisStorageClass, config interface{}) (*corev1.ConfigMap, error) {
	clusterConfig, err := newNodeConfigString(cluster, storage)
	if err != nil {
		return nil, err
	}

	protocolConfig, err := newProtocolConfigString(config)
	if err != nil {
		return nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    cluster.Labels,
		},
		Data: map[string]string{
			clusterConfigFile:  clusterConfig,
			protocolConfigFile: protocolConfig,
		},
	}, nil
}

// getClusterResourceName returns the given resource name for the given cluster
func getClusterResourceName(cluster *v1beta2.Cluster, resource string) string {
	return fmt.Sprintf("%s-%s", cluster.Name, resource)
}

// GetClusterHeadlessServiceName returns the headless service name for the given cluster
func GetClusterHeadlessServiceName(cluster *v1beta2.Cluster) string {
	return getClusterResourceName(cluster, headlessServiceSuffix)
}

// getPodName returns the name of the pod for the given pod ID
func getPodName(cluster *v1beta2.Cluster, pod int) string {
	return fmt.Sprintf("%s-%d", cluster.Name, pod)
}

// getPodDNSName returns the fully qualified DNS name for the given pod ID
func getPodDNSName(cluster *v1beta2.Cluster, pod int) string {
	return fmt.Sprintf("%s-%d.%s.%s.svc.cluster.local", cluster.Name, pod, GetClusterHeadlessServiceName(cluster), cluster.Namespace)
}

// GetDatabaseFromClusterAnnotations returns the database name from the given cluster annotations
func GetDatabaseFromClusterAnnotations(cluster *v1beta2.Cluster) (string, error) {
	database, ok := cluster.Annotations[databaseAnnotation]
	if !ok {
		return "", errors.New("cluster missing database annotation")
	}
	return database, nil
}

// GetClusterIDFromClusterAnnotations returns the cluster ID from the given cluster annotations
func GetClusterIDFromClusterAnnotations(cluster *v1beta2.Cluster) (int32, error) {
	idstr, ok := cluster.Annotations[clusterAnnotation]
	if !ok {
		return 1, nil
	}

	id, err := strconv.ParseInt(idstr, 0, 32)
	if err != nil {
		return 0, err
	}
	return int32(id), nil
}

// newNodeConfigString creates a node configuration string for the given cluster
func newNodeConfigString(cluster *v1beta2.Cluster, storage *storage.RedisStorageClass) (string, error) {
	clusterID, err := GetClusterIDFromClusterAnnotations(cluster)
	if err != nil {
		return "", err
	}

	clusterDatabase, err := GetDatabaseFromClusterAnnotations(cluster)
	if err != nil {
		return "", err
	}

	members := make([]*api.MemberConfig, storage.Spec.Backend.Replicas)
	for i := 0; i < int(storage.Spec.Backend.Replicas); i++ {
		members[i] = &api.MemberConfig{
			ID:      getPodName(cluster, i),
			Host:    getPodDNSName(cluster, i),
			APIPort: apiPort,
		}
	}

	partitions := make([]*api.PartitionId, 0, cluster.Spec.Partitions)
	for partitionID := (cluster.Spec.Partitions * (clusterID - 1)) + 1; partitionID <= cluster.Spec.Partitions*clusterID; partitionID++ {
		partition := &api.PartitionId{
			Partition: partitionID,
			Cluster: &api.ClusterId{
				ID: int32(clusterID),
				DatabaseID: &api.DatabaseId{
					Name:      clusterDatabase,
					Namespace: cluster.Namespace,
				},
			},
		}
		partitions = append(partitions, partition)
	}

	config := &api.ClusterConfig{
		Members:    members,
		Partitions: partitions,
	}

	marshaller := jsonpb.Marshaler{}
	return marshaller.MarshalToString(config)
}

// newProtocolConfigString creates a protocol configuration string for the given cluster and protocol
func newProtocolConfigString(config interface{}) (string, error) {
	bytes, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// NewClusterService returns a new service for a cluster
func NewClusterService(cluster *v1beta2.Cluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    cluster.Labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "api",
					Port: 5678,
				},
			},
			Selector: cluster.Labels,
		},
	}
}

// NewClusterHeadlessService returns a new headless service for a cluster group
func NewClusterHeadlessService(cluster *v1beta2.Cluster) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetClusterHeadlessServiceName(cluster),
			Namespace: cluster.Namespace,
			Labels:    cluster.Labels,
			Annotations: map[string]string{
				"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "api",
					Port: 5678,
				},
			},
			PublishNotReadyAddresses: true,
			ClusterIP:                "None",
			Selector:                 cluster.Labels,
		},
	}
}

// NewStatefulSet returns a new StatefulSet for a cluster group
func NewStatefulSet(cluster *v1beta2.Cluster, storage *storage.RedisStorageClass) (*appsv1.StatefulSet, error) {
	volumes := []corev1.Volume{
		newConfigVolume(cluster.Name),
	}

	volumes = append(volumes, newDataVolume())
	image := storage.Spec.Backend.Image
	pullPolicy := storage.Spec.Backend.ImagePullPolicy
	readinessProbe :=
		&corev1.Probe{
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{
					Command: []string{"redis-cli"},
				},
			},
			InitialDelaySeconds: 5,
			TimeoutSeconds:      10,
			FailureThreshold:    12,
		}
	livenessProbe :=
		&corev1.Probe{
			Handler: corev1.Handler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.IntOrString{Type: intstr.Int, IntVal: probePort},
				},
			},
			InitialDelaySeconds: 60,
			TimeoutSeconds:      10,
		}

	if pullPolicy == "" {
		pullPolicy = corev1.PullIfNotPresent
	}

	apiContainerPort := corev1.ContainerPort{
		Name:          "api",
		ContainerPort: apiPort,
	}

	containerBuilder := NewContainer()
	container := containerBuilder.SetImage(image).
		SetName(cluster.Name).
		SetPullPolicy(pullPolicy).
		SetPorts([]corev1.ContainerPort{apiContainerPort}).
		SetReadinessProbe(readinessProbe).
		SetLivenessProbe(livenessProbe).
		SetVolumeMounts([]corev1.VolumeMount{newDataVolumeMount(), newConfigVolumeMount()}).
		Build()

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    cluster.Labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: GetClusterHeadlessServiceName(cluster),
			Replicas:    &storage.Spec.Backend.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: cluster.Labels,
			},
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,

			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: cluster.Labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						newContainer(container),
					},
					Volumes: volumes,
				},
			},
		},
	}, nil
}

func (r *Reconciler) addService(cluster *v1beta2.Cluster, storage *v1beta1.RedisStorageClass) error {
	log.Info("Creating raft service", "Name", cluster.Name, "Namespace", cluster.Namespace)
	service := NewClusterService(cluster)
	if err := controllerutil.SetControllerReference(storage, service, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), service)

}

func (r *Reconciler) addHeadlessService(cluster *v1beta2.Cluster, storage *v1beta1.RedisStorageClass) error {
	log.Info("Creating headless raft service", "Name", cluster.Name, "Namespace", cluster.Namespace)
	service := NewClusterHeadlessService(cluster)
	if err := controllerutil.SetControllerReference(storage, service, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), service)
}

func (r *Reconciler) addStatefulSet(cluster *v1beta2.Cluster, storage *v1beta1.RedisStorageClass) error {
	log.Info("Creating raft replicas", "Name", cluster.Name, "Namespace", cluster.Namespace)
	set, err := NewStatefulSet(cluster, storage)
	if err != nil {
		return err
	}
	if err := controllerutil.SetControllerReference(storage, set, r.scheme); err != nil {
		return err
	}

	return r.client.Create(context.TODO(), set)
}

func (r *Reconciler) addConfigMap(cluster *v1beta2.Cluster, storage *v1beta1.RedisStorageClass) error {
	log.Info("Creating raft ConfigMap", "Name", cluster.Name, "Namespace", cluster.Namespace)
	var config interface{}

	cm, err := NewClusterConfigMap(cluster, storage, config)
	if err != nil {
		return err
	}
	if err := controllerutil.SetControllerReference(storage, cm, r.scheme); err != nil {
		return err
	}
	return r.client.Create(context.TODO(), cm)
}
