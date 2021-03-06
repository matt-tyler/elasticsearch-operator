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

func newMasterDeployment(cluster *esV1.Cluster, serviceURL string) *v1beta2.Deployment {
	replicas := int32(1)
	minimumNodes := string((replicas + 1) / 2)
	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"role": "master",
		},
	}

	labels := map[string]string{}
	for k, v := range cluster.Labels {
		metav1.AddLabelToSelector(&selector, k, v)
		labels[k] = v
	}
	labels["operator"] = "elasticsearch-operator"

	deployment := &v1beta2.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%v-master-deployment", cluster.Name),
			Labels: labels,
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
						Env: []v1.EnvVar{
							v1.EnvVar{"cluster.name", cluster.Name, nil},
							v1.EnvVar{"network.host", "$${HOSTNAME}", nil},
							v1.EnvVar{"boostrap.memory_lock", "true", nil},
							v1.EnvVar{"node.master", "true", nil},
							v1.EnvVar{"node.data", "false", nil},
							v1.EnvVar{"discovery.zen.ping.unicast.hosts", serviceURL, nil},
							v1.EnvVar{"discovery.zen.minimum_master_nodes", minimumNodes, nil},
						},
					}},
				},
			},
		},
	}
	return deployment
}

// return a headless service for master discovery
func newMasterService(cluster *esV1.Cluster) *v1.Service {
	selector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			"role": "master",
		},
	}
	labels := map[string]string{}
	for k, v := range cluster.Labels {
		metav1.AddLabelToSelector(&selector, k, v)
		labels[k] = v
	}
	labels["operator"] = "elasticsearch-operator"

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("%v-master-service", cluster.Name),
			Labels: labels,
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
			Selector:  selector.MatchLabels,
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
