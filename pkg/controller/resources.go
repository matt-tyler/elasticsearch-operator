package controller

import (
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

// New creates a new deployment of a cluster
func New(c *rest.RESTClient, objectMeta metav1.ObjectMeta) error {
	deployment := newMasterDeployment(objectMeta)
	deploymentCopy := deployment.DeepCopy()

	service := newMasterService(objectMeta)
	serviceCopy := service.DeepCopy()

	deploymentClient := v1beta1.New(c).Deployments(objectMeta.Namespace)
	_, err := deploymentClient.Create(deploymentCopy)
	if err != nil {
		return err
	}

	serviceClient := corev1.New(c).Services(objectMeta.Namespace)
	_, err = serviceClient.Create(serviceCopy)
	if err != nil {
		return err
	}
	return nil
}

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
func newMasterService(objectMeta metav1.ObjectMeta) *v1.Service {
	service := &v1.Service{
		ObjectMeta: objectMeta,
		Spec: v1.ServiceSpec{
			Type:      "ClusterIP",
			ClusterIP: "None",
			Selector:  nil,
			Ports: []v1.ServicePort{{
				Name: "",
				Port: 9200,
			}, {
				Name: "",
				Port: 9300,
			}},
		},
	}
	return service
}

// Default Pod Template
// take template and create master and data nodes

// statefulset for data nodes
