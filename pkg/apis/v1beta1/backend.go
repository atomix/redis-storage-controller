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

import corev1 "k8s.io/api/core/v1"

// Backend is a proxy configuration
type Backend struct {

	// Replicas is the number of raft replicas
	Replicas int32 `json:"replicas,omitempty"`
	// Image is the image to run
	Image string `json:"image,omitempty"`

	// ImagePullPolicy is the pull policy to apply
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

// BackendStatus is the cluster proxy status
type BackendStatus struct {
	// Ready indicates whether the proxy is ready
	Ready bool `json:"ready,omitempty"`
}
