package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SimpleAppSpec defines the desired state of SimpleApp
type SimpleAppSpec struct {
	// +optional
	//下面三个值的定义定义写在了yaml文件
	Replicas *int32 `json:"replicas,omitempty"` //就是pod数量，后面的replicas,omitempty用来区分有没有写这个值
	Image    string `json:"image"`
	Port     int32  `json:"port"`
	// +optional
	ServiceType corev1.ServiceType `json:"serviceType,omitempty"`
}

// SimpleAppStatus defines the observed state of SimpleApp
type SimpleAppStatus struct {
	// 实际跑起来的pod
	Replicas int32 `json:"replicas,omitempty"`
	// 有多少个pod是就绪的
	Ready int32 `json:"ready,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".status.replicas"
//+kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.ready"
//+kubebuilder:printcolumn:name="Image",type="string",JSONPath=".spec.image"

// SimpleApp is the Schema for the simpleapps API
type SimpleApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SimpleAppSpec   `json:"spec,omitempty"`
	Status            SimpleAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type SimpleAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SimpleApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SimpleApp{}, &SimpleAppList{})
}
