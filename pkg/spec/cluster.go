package spec

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ResourcePlural = "clusters"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ClusterSpec   `json:"spec"`
	Status            ClusterStatus `json"status,omitempty"`
}

type ClusterSpec struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

type ClusterStatus struct {
	state ClusterState `json:"state,omitempty"`
}

type ClusterState string

const (
	ClusterStateCreated  ClusterState = "Created"
	ClusterStateUpdating ClusterState = "Updating"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterList struct {
	metav1.TypeMeta `json:,inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Cluster `json:"items"`
}
