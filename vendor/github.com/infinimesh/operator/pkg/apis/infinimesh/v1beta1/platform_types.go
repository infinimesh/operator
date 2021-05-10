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
	MQTT                     PlatformMQTTBroker               `json:"mqtt,omitempty" protobuf:"bytes,1,name=mqtt"`
	DGraph                   PlatformDgraph                   `json:"dgraph,omitempty" protobuf:"bytes,2,name=dgraph"`
	DGraphAlpha              PlatformDgraphAlpha              `json:"dgraphAlpha,omitempty" protobuf:"bytes,2,name=dgraphAlpha"`
	DGraphZero               PlatformDgraphZero               `json:"dgraphZero,omitempty" protobuf:"bytes,2,name=dgraphZero"`
	Kafka                    PlatformKafka                    `json:"kafka,omitempty" protobuf:"bytes,2,name=kafka"`
	Apiserver                PlatformApiserver                `json:"apiserver,omitempty" protobuf:"bytes,3,name=apiserver"`
	App                      PlatformApp                      `json:"app,omitempty" protobuf:"bytes,4,name=app"`
	InfinimeshDefaultStorage PlatformInfinimeshDefaultStorage `json:"infinimeshDefaultStorage,omitempty" protobuf:"bytes,2,name=infinimeshDefaultStorage"`
	Controller               PlatformController               `json:"controller,omitempty" protobuf:"bytes,13,name=controller"`
	Host                     PlatformHost                     `json:"host,omitempty" protobuf:"bytes,1,name=host"`

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

type PlatformDgraph struct {
	Storage *core.PersistentVolumeClaimSpec `json:"storage,omitempty" protobuf:"bytes,1,name=storage"`
}
type PlatformDgraphAlpha struct {
	Storage *core.PersistentVolumeClaimSpec `json:"storage,omitempty" protobuf:"bytes,1,name=storage"`
}
type PlatformDgraphZero struct {
	Storage *core.PersistentVolumeClaimSpec `json:"storage,omitempty" protobuf:"bytes,1,name=storage"`
}
type PlatformInfinimeshDefaultStorage struct {
	Storage *core.PersistentVolumeClaimSpec `json:"storage,omitempty" protobuf:"bytes,1,name=storage"`
}
type PlatformController struct {
	DeviceDetails              bool `json:"device_details,omitempty" protobuf:"bytes,1,name=device_details"`
	APIServer                  bool `json:"apiserver,omitempty" protobuf:"bytes,1,name=apiserver"`
	DeviceRegistry             bool `json:"device_registry,omitempty" protobuf:"bytes,1,name=device_registry"`
	Dgraph                     bool `json:"dgraph,omitempty" protobuf:"bytes,1,name=dgraph"`
	Frontend                   bool `json:"frontend,omitempty" protobuf:"bytes,1,name=frontend"`
	HardDeleteNamespaceCronjob bool `json:"hard_delete_namespace_cronjob,omitempty" protobuf:"bytes,1,name=hard_delete_namespace_cronjob"`
	Timeseries                 bool `json:"timeseries,omitempty" protobuf:"bytes,1,name=timeseries"`
	MQTTBridge                 bool `json:"mqtt_bridge,omitempty" protobuf:"bytes,1,name=mqtt_bridge"`
	NodeServer                 bool `json:"nodeserver,omitempty" protobuf:"bytes,1,name=nodeserver"`
	ResetRootAccountPwd        bool `json:"reset_root_account_pwd,omitempty" protobuf:"bytes,1,name=reset_root_account_pwd"`
	TelemetryRouter            bool `json:"telemetry-router,omitempty" protobuf:"bytes,1,name=telemetry-router"`
	Twin                       bool `json:"twin,omitempty" protobuf:"bytes,1,name=twin"`
	APIServerRest              bool `json:"apiserver_rest,omitempty" protobuf:"bytes,1,name=apiserver_rest"`
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
type PlatformHost struct {
	Registry string `json:"registry,omitempty" protobuf:"bytes,1,name=registry"`
	Repo     string `json:"repo,omitempty" protobuf:"bytes,1,name=repo"`
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
