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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type DeploymentStrategySpec struct {
	// Typ der Strategie: RollingUpdate oder Recreate
	// +kubebuilder:validation:Enum=RollingUpdate;Recreate
	// +optional
	Type string `json:"type,omitempty"`

	// RollingUpdate-Parameter
	// +optional
	RollingUpdate *RollingUpdateSpec `json:"rollingUpdate,omitempty"`
}

type RollingUpdateSpec struct {
	// Maximale Anzahl Pods über Soll
	// +optional
	MaxSurge *int32 `json:"maxSurge,omitempty"`

	// Maximale Anzahl nicht verfügbarer Pods
	// +optional
	MaxUnavailable *int32 `json:"maxUnavailable,omitempty"`
}

type ResourceSpec struct {
	// Mindest-Ressourcen
	// +optional
	Requests *ResourceValues `json:"requests,omitempty"`

	// Maximale Ressourcen
	// +optional
	Limits *ResourceValues `json:"limits,omitempty"`
}

type ResourceValues struct {
	// CPU, z.B. "100m", "1"
	// +optional
	// +kubebuilder:validation:Pattern=`^([0-9]+m|[0-9]+(\.[0-9]+)?)$`
	CPU string `json:"cpu,omitempty"`

	// Memory, z.B. "128Mi", "1Gi"
	// +optional
	// +kubebuilder:validation:Pattern=`^[0-9]+(Ki|Mi|Gi|Ti)$`
	Memory string `json:"memory,omitempty"`
}

type DeploymentSpec struct {
	// Name des Deployments
	Name string `json:"name"`

	// Anzahl Replikas
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Deployment-Strategie
	// +optional
	Strategy *DeploymentStrategySpec `json:"strategy,omitempty"`

	// Ressourcen-Vorgaben
	// +optional
	Resources *ResourceSpec `json:"resources,omitempty"`

	// Container dieses Deployments
	Containers []ContainerSpec `json:"containers"`
}

type ReadinessProbeSpec struct {
	// +optional
	HTTPGet *HTTPGetProbeSpec `json:"httpGet,omitempty"`

	// +optional
	InitialDelaySeconds *int32 `json:"initialDelaySeconds,omitempty"`

	// +optional
	PeriodSeconds *int32 `json:"periodSeconds,omitempty"`
}

type HTTPGetProbeSpec struct {
	Path string `json:"path"`
	Port int32  `json:"port"`

	// +optional
	Scheme *corev1.URIScheme `json:"scheme,omitempty"`
}

type ContainerSpec struct {
	Name  string `json:"name"`
	Image string `json:"image"`

	// +optional
	ReadinessProbe *ReadinessProbeSpec `json:"readinessProbe,omitempty"`

	// +optional
	Resources *ResourceSpec `json:"resources",omitempty"`
}

// DemoAppSpec defines the desired state of DemoApp
type DemoAppSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html
	// foo is an example field of DemoApp. Edit demoapp_types.go to remove/update
	// +optional
	Foo *string `json:"foo,omitempty"`

	// Liste der Deployments
	Deployments []DeploymentSpec `json:"deployments"`
}

// DemoAppStatus defines the observed state of DemoApp.
type DemoAppStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the DemoApp resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions        []metav1.Condition `json:"conditions,omitempty"`
	AvailableReplicas int32              `json:"availableReplicas,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=gateways/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.example.com,resources=demoapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.example.com,resources=demoapps/status,verbs=get;update;patch


// DemoApp is the Schema for the demoapps API
type DemoApp struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of DemoApp
	// +required
	Spec DemoAppSpec `json:"spec"`

	// status defines the observed state of DemoApp
	// +optional
	Status DemoAppStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// DemoAppList contains a list of DemoApp
type DemoAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []DemoApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DemoApp{}, &DemoAppList{})
}
