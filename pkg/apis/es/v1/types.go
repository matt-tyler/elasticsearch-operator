package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ResourcePlural = "clusters"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ClusterSpec `json:"spec"`
}

type ClusterSpec struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterList struct {
	metav1.TypeMeta `json:"inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Cluster `json:"items"`
}
