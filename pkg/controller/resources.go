package controller

import (
	"fmt"

	esV1 "github.com/matt-tyler/elasticsearch-operator/pkg/apis/es/v1"
	v1beta2 "k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func newMasterDeployment(cluster *esV1.Cluster) *v1beta2.Deployment {
	replicas := int32(1)
	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"role": "master",
		},
	}

	for k, v := range cluster.Labels {
		metav1.AddLabelToSelector(&selector, k, v)
	}

	deployment := &v1beta2.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%v-master-deployment", cluster.Name),
			Labels: cluster.Labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, schema.GroupVersionKind{
					Group:   esV1.SchemeGroupVersion.Group,
					Version: esV1.SchemeGroupVersion.Version,
					Kind:    "Cluster",
				}),
			},
		},
		Spec: v1beta2.DeploymentSpec{
			Strategy: v1beta2.DeploymentStrategy{
				Type: v1beta2.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &v1beta2.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				},
			},
			Replicas: &replicas,
			Selector: &selector,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: selector.MatchLabels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:            "elastic-master",
						Image:           "docker.elastic.co/elasticsearch/elasticsearch-oss:6.1.1",
						ImagePullPolicy: v1.PullIfNotPresent,
						Ports: []v1.ContainerPort{{
							ContainerPort: 9200,
						}, {
							ContainerPort: 9300,
						}},
					}},
				},
			},
		},
	}
	return deployment
}

// return a headless service for master discovery
func newMasterService(cluster *esV1.Cluster) *v1.Service {
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%v-master-service", cluster.Name),
			Labels: cluster.Labels,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cluster, schema.GroupVersionKind{
					Group:   esV1.SchemeGroupVersion.Group,
					Version: esV1.SchemeGroupVersion.Version,
					Kind:    "Cluster",
				}),
			},
		},
		Spec: v1.ServiceSpec{
			Type:      "ClusterIP",
			ClusterIP: "None",
			Selector:  nil,
			Ports: []v1.ServicePort{{
				Name: "rest",
				Port: 9200,
			}, {
				Name: "node",
				Port: 9300,
			}},
		},
	}
	return service
}

// Default Pod Template
// take template and create master and data nodes

// statefulset for data nodes
