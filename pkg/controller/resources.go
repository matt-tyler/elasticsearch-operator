package controller

import (
	"fmt"

	esV1 "github.com/matt-tyler/elasticsearch-operator/pkg/apis/es/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func newMasterDeployment(objectMeta metav1.ObjectMeta) *appsv1beta1.Deployment {
	replicas := new(int32)
	deployment := &appsv1beta1.Deployment{
		ObjectMeta: objectMeta,
		Spec: appsv1beta1.DeploymentSpec{
			Strategy: appsv1beta1.DeploymentStrategy{
				Type: appsv1beta1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1beta1.RollingUpdateDeployment{
					MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
					MaxSurge:       &intstr.IntOrString{Type: intstr.Int, IntVal: 1},
				},
			},
			Replicas: replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:            "elastic-master",
						Image:           "",
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
			Name: fmt.Sprintf("%v-master-service", cluster.Name),
			Labels: map[string]string{
				"elasticsearch-cluster": cluster.Name,
			},
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
