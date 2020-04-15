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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RedisStorageClassGroup cache storage class group
const RedisStorageClassGroup = "storage.cloud.atomix.io"

// RedisStorageClassVersion cache storage class version
const RedisStorageClassVersion = "v1beta1"

// RedisStorageClassKind cache storage class kind
const RedisStorageClassKind = "RedisStorageClass"

// RedisStorageClassSpec defines the desired state of RedisStorage
type RedisStorageClassSpec struct {

	// Proxy is the redis proxy
	Proxy Proxy `json:"proxy,omitempty"`

	Backend Backend `json:"backend,omitempty"`

	RedisStorageStatus RedisStorageClassStatus `json:"redis,omitempty"`
}

// RedisStorageClassStatus defines the observed state of RedisStorage
type RedisStorageClassStatus struct {

	// Proxy is the proxy status
	Proxy *ProxyStatus `json:"proxy,omitempty"`

	Backend *BackendStatus `json:"backend,omitempty"`
}

// +kubebuilder:object:root=true

// RedisStorage is the Schema for the redisstoragess API
type RedisStorageClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisStorageClassSpec   `json:"spec,omitempty"`
	Status RedisStorageClassStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RedisStorageClassList contains a list of RedisStorage
type RedisStorageClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RedisStorageClass `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisStorageClass{}, &RedisStorageClassList{})
}
