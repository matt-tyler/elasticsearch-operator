package client

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/matt-tyler/elasticsearch-operator/pkg/log"
	spec "github.com/matt-tyler/elasticsearch-operator/pkg/spec"
	apiv1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
)

func PrettyJson(v interface{}) string {
	logger := log.NewLogger()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		logger.Panicf("%v", err)
	}
	return string(b)
}

const crdName = spec.ResourcePlural + "." + spec.GroupName

func CreateCustomResourceDefinition(clientset apiextensionsclient.Interface) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	logger := log.NewLogger()
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   spec.GroupName,
			Version: spec.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: spec.ResourcePlural,
				Kind:   reflect.TypeOf(spec.Cluster{}).Name(),
			},
		},
	}

	logger.Debugf("Creating custom resource:\n%v", PrettyJson(crd))

	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		logger.Debugf("Failed to create custom resource")
		return nil, err
	}

	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crdName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					logger.Errorf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})

	if err != nil {
		deleteErr := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crdName, nil)
		if deleteErr != nil {
			return nil, errors.NewAggregate([]error{err, deleteErr})
		}
		return nil, err
	}
	return crd, nil
}

func WaitFor(client *rest.RESTClient, name string) error {
	return wait.Poll(100*time.Millisecond, 10*time.Second, func() (bool, error) {
		var cluster spec.Cluster
		err := client.Get().
			Resource(spec.ResourcePlural).
			Namespace(apiv1.NamespaceDefault).
			Name(name).
			Do().Into(&cluster)

		if err == nil && cluster.Status.State == spec.ClusterStateCreated {
			return true, nil
		}

		return false, err
	})
}
