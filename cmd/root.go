package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	. "github.com/matt-tyler/elasticsearch-operator/pkg/client"
	. "github.com/matt-tyler/elasticsearch-operator/pkg/controller"
	"github.com/matt-tyler/elasticsearch-operator/pkg/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
