package cmd

import (
	"context"
	"fmt"
	"github.com/matt-tyler/elasticsearch-operator/pkg/client"
	"github.com/matt-tyler/elasticsearch-operator/pkg/controller"
	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"os"
	"os/signal"
	"syscall"
)

var RootCmd = &cobra.Command{
	Use:   "elasticsearch-operator",
	Short: "An elasticsearch operator for Kubernetes",
	Run: func(cmd *cobra.Command, args []string) {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM)

		clientConfig, err := rest.InClusterConfig()
		if err != nil {
			panic(err)
		}

		//apiextensionsclientset, err := apiextensionsclient.NewForConfig(clientConfig)
		//if err != nil {
		//    panic(err)
		//}

		client, scheme, err := client.NewClient(clientConfig)
		if err != nil {
			panic(err)
		}

		controller := controller.NewController(client, scheme)

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

}
