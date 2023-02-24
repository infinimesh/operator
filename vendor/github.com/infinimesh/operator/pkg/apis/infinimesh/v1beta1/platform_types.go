/*
Copyright 2019 infinimesh, inc.

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

package v1beta1

import (
	core "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PlatformSpec defines the desired state of Platform
type PlatformSpec struct {
	MQTT      PlatformMQTTBroker `json:"mqtt,omitempty" protobuf:"bytes,1,name=mqtt"`
	DGraph    PlatformDgraph     `json:"dgraph,omitempty" protobuf:"bytes2,name=dgraph"`
	Kafka     PlatformKafka      `json:"kafka,omitempty" protobuf:"bytes,2,name=kafka"`
	Apiserver PlatformApiserver  `json:"apiserver,omitempty" protobuf:"bytes,3,name=apiserver"`
	App       PlatformApp        `json:"app,omitempty" protobuf:"bytes,4,name=app"`

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

type PlatformDgraph struct {
	Storage *core.PersistentVolumeClaimSpec `json:"storage,omitempty" protobuf:"bytes,1,name=storage"`
}

type PlatformTimeseries struct {
	TimescaleDB *PlatformTimescaleDB `json:"timescaledb,omitempty" protobuf:"bytes,1,name=timescaledb"`
}

type PlatformTimescaleDB struct {
	Storage *core.PersistentVolumeClaimSpec `json:"storage,omitempty" protobuf:"bytes,2,name=storage"`
}

type PlatformApp struct {
	Host string                         `json:"host,omitempty" protobuf:"bytes,1,name=host"`
	TLS  []extensionsv1beta1.IngressTLS `json:"tls,omitempty" protobuf:"bytes,2,name=tls"`
}

type PlatformApiserver struct {
	GRPC    PlatformGRPCApiserver    `json:"grpc,omitempty" protobuf:"bytes,1,name=grpc"`
	Restful PlatformRestfulApiserver `json:"restful,omitempty" protobuf:"bytes,2,name=restful"`
}

type PlatformRestfulApiserver struct {
	Host string                         `json:"host,omitempty" protobuf:"bytes,1,name=host"`
	TLS  []extensionsv1beta1.IngressTLS `json:"tls,omitempty" protobuf:"bytes,2,name=tls"`
}
type PlatformGRPCApiserver struct {
	Host string                         `json:"host,omitempty" protobuf:"bytes,1,name=host"`
	TLS  []extensionsv1beta1.IngressTLS `json:"tls,omitempty" protobuf:"bytes,2,name=tls"`
}

type PlatformKafka struct {
	BootstrapServers string `json:"bootstrapServers,omitempty" protobuf:"bytes,1,name=bootstrapServers"`
}

type PlatformMQTTBroker struct {
	SecretName string `json:"secretName,omitempty" protobuf:"bytes,1,name=secretName"`
}

// PlatformStatus defines the observed state of Platform
type PlatformStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Platform is the Schema for the platforms API
// +k8s:openapi-gen=true
type Platform struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlatformSpec   `json:"spec,omitempty"`
	Status PlatformStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PlatformList contains a list of Platform
type PlatformList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Platform `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Platform{}, &PlatformList{})
}
