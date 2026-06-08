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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RedisClusterSpec defines the desired state of RedisCluster
type RedisClusterSpec struct {
	// Size define a quantidade de nós Redis (1 Master + N Replicas). Padrão recomendado: 3
	// +kubebuilder:validation:Minimum=1
	Size int32 `json:"size"`

	// SentinelSize define a quantidade de nós do Sentinel para o quórum. Padrão recomendado: 3
	// +kubebuilder:validation:Minimum=3
	SentinelSize int32 `json:"sentinelSize"`

	// RedisImage permite customizar a versão da imagem do Redis
	// +kubebuilder:default:="redis:7.0-alpine"
	RedisImage string `json:"redisImage,omitempty"`
}

// RedisClusterStatus defines the observed state of RedisCluster
type RedisClusterStatus struct {
	// Phase representa o status atual do cluster (ex: Creating, Ready, Degraded)
	Phase string `json:"phase,omitempty"`

	// MasterNode guarda o nome do pod que atualmente é o Master (útil para debug e roteamento)
	MasterNode string `json:"masterNode,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RedisCluster is the Schema for the redisclusters API
type RedisCluster struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of RedisCluster
	// +required
	Spec RedisClusterSpec `json:"spec"`

	// status defines the observed state of RedisCluster
	// +optional
	Status RedisClusterStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// RedisClusterList contains a list of RedisCluster
type RedisClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []RedisCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RedisCluster{}, &RedisClusterList{})
}
