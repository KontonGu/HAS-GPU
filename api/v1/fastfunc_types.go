/*
Copyright 2024 HAS-Function Authors, KontonGu (Jianfeng Gu), et. al.

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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FaSTFuncSpec defines the desired state of FaSTFunc
type FaSTFuncSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +optional
	PodSpec corev1.PodSpec `json:"podSpec,omitempty"`

	// Selector is a label query over pods that should match the replica count.
	// Label keys and values that must match in order to be controlled by this replica set.
	// It must match the pod template's labels.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	Selector *metav1.LabelSelector `json:"selector" protobuf:"bytes,2,opt,name=selector"`
}

// FaSTFuncStatus defines the observed state of FaSTFunc
type FaSTFuncStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:rbac:groups=*,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=*,resources=fastpods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=fastgshare.caps.in.tum,resources=fastpods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=fastgshare.caps.in.tum,resources=*,verbs=*

// FaSTFunc is the Schema for the fastfuncs API
// +kubebuilder:subresource:status
type FaSTFunc struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FaSTFuncSpec   `json:"spec,omitempty"`
	Status FaSTFuncStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FaSTFuncList contains a list of FaSTFunc
type FaSTFuncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FaSTFunc `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FaSTFunc{}, &FaSTFuncList{})
}
