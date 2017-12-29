package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/matt-tyler/elasticsearch-operator/pkg/apis/es"
	esV1 "github.com/matt-tyler/elasticsearch-operator/pkg/apis/es/v1"
	. "github.com/matt-tyler/elasticsearch-operator/pkg/controller"
	"github.com/matt-tyler/elasticsearch-operator/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

const crdName = esV1.ResourcePlural + "." + es.GroupName

func CreateCustomResourceDefinition(clientset apiextensionsclient.Interface) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	logger := log.NewLogger()
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   es.GroupName,
			Version: esV1.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: esV1.ResourcePlural,
				Kind:   reflect.TypeOf(esV1.Cluster{}).Name(),
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

func PrettyJson(v interface{}) string {
	logger := log.NewLogger()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		logger.Panicf("%v", err)
	}
	return string(b)
}

var RootCmd = &cobra.Command{
	Use:   "elasticsearch-operator",
	Short: "An elasticsearch operator for Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		logger := log.NewLogger()
		defer logger.Infof("Elasticsearch Operator has stopped")

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

		kubeconfig := viper.GetString("kubeconfig")

		clientConfig, err := buildConfig(kubeconfig)
		if err != nil {
			logger.Panicf("%v", err)
		}

		apiextensionsclientset, err := apiextensionsclient.NewForConfig(clientConfig)
		if err != nil {
			logger.Panicf("%v", err)
		}

		crd, err := CreateCustomResourceDefinition(apiextensionsclientset)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			logger.Panicf("%v", err)
		}

		if crd != nil {
			defer func() {
				logger.Infof("Removing custom resource definition from cluster...")
				if err := apiextensionsclientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.Name, nil); err != nil {
					logger.Errorf("Failed to remove custom resource definition")
				} else {
					logger.Infof("Custom resource definition removed.")
				}
			}()
		}

		controller := NewController(clientConfig)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go controller.Run(ctx)

		select {
		case <-sigs:
			logger.Infof("Elasticsearch Operator is stopping...")
			return
		}
	},
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringP("kubeconfig", "f", "", "Path to kubeconfig")
	viper.BindPFlag("kubeconfig", RootCmd.PersistentFlags().Lookup("kubeconfig"))
}
