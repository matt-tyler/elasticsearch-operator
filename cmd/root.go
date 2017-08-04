package cmd

import (
	"context"
	"fmt"
	. "github.com/matt-tyler/elasticsearch-operator/pkg/client"
	. "github.com/matt-tyler/elasticsearch-operator/pkg/controller"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/signal"
	"syscall"
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
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM)

		kubeconfig := viper.GetString("kubeconfig")

		clientConfig, err := buildConfig(kubeconfig)
		if err != nil {
			panic(err)
		}

		apiextensionsclientset, err := apiextensionsclient.NewForConfig(clientConfig)
		if err != nil {
			panic(err)
		}

		crd, err := CreateCustomResourceDefinition(apiextensionsclientset)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			panic(err)
		}

		if crd != nil {
			defer apiextensionsclientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.Name, nil)
		}

		client, scheme, err := NewClient(clientConfig)
		if err != nil {
			panic(err)
		}

		controller := NewController(client, scheme)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go controller.Run(ctx)

		select {
		case <-sigs:
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
