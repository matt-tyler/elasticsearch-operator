// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package e2e

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/matt-tyler/elasticsearch-operator/e2e/suite"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/matt-tyler/elasticsearch-operator/e2e/gke"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var Kubeconfig string
var Image string
var Up bool
var Down bool
var Test bool

func init() {
	flag.StringVar(&Kubeconfig, "kubeconfig", "", "Location of kubeconfig")
	flag.StringVar(&Image, "image", "gcr.io/schnauzer-163208/elasticsearch-operator:latest", "image under test")
	flag.BoolVar(&Up, "up", false, "")
	flag.BoolVar(&Down, "down", false, "")
	flag.BoolVar(&Test, "test", false, "")
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}

func RunE2ETests(t *testing.T) {
	flag.Parse()

	clusterID := "e2e-test-cluster"
	ctx := context.Background()
	client := gke.GkeClient{}

	var config *rest.Config

	if Up || Down {
		if err := gke.NewGkeClient(&client, ctx, os.Getenv("PROJECT"), os.Getenv("ZONE")); err != nil {
			panic(err)
		}
	}

	if Up {
		createCluster(client, clusterID)
		cluster, err := client.GetCluster(clusterID)
		if err != nil {
			panic(err)
		}

		config, err = cluster.Config()
		if err != nil {
			panic(err)
		}

		configAccess := clientcmd.NewDefaultClientConfigLoadingRules()
		clientcmd.ModifyConfig(configAccess, *(clientConfig(cluster)), false)
	}

	if Down {
		defer deleteCluster(client, clusterID)
	}

	if !Test {
		return
	}

	if config == nil {
		var err error
		config, err = buildConfig(Kubeconfig)
		if err != nil {
			panic(err)
		}
	}

	suite.Setup(config, Image)

	RegisterFailHandler(Fail)

	r := make([]Reporter, 0)

	RunSpecsWithDefaultAndCustomReporters(t, "Elasticsearch Operator E2E Suite", r)
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func deleteCluster(client gke.GkeClient, clusterID string) {
	fmt.Printf("Deleting Cluster: %v\n", clusterID)
	op, err := client.DeleteCluster(clusterID)
	if err != nil {
		panic(err)
	}
	client.Done(op)
	fmt.Printf("Cluster %v deleted\n", clusterID)
}

func createCluster(client gke.GkeClient, clusterID string) {
	fmt.Printf("Creating cluster: %v\n", clusterID)
	op, err := client.CreateCluster(clusterID)
	if err != nil {
		panic(err)
	}
	client.Done(op)
	fmt.Printf("Cluster %v created\n", clusterID)
}

func clientConfig(gc *gke.Cluster) *clientcmdapi.Config {
	gcloudPath, err := exec.LookPath("gcloud")
	if err != nil {
		panic(err)
	}

	config := clientcmdapi.NewConfig()

	cluster := clientcmdapi.NewCluster()

	caCert, err := base64.StdEncoding.DecodeString(gc.Auth.ClusterCaCertificate)
	if err != nil {
		panic(err)
	}

	cluster.CertificateAuthorityData = []byte(caCert)
	cluster.Server = fmt.Sprintf("https://%v", gc.Endpoint)

	context := clientcmdapi.NewContext()
	context.Cluster = "e2e-test-cluster"
	context.AuthInfo = "e2e-test-cluster-user"

	authInfo := clientcmdapi.NewAuthInfo()
	authInfo.AuthProvider = &clientcmdapi.AuthProviderConfig{
		Name: "gcp",
		Config: map[string]string{
			"cmd-args":   "config config-helper --format=json",
			"cmd-path":   gcloudPath,
			"expiry-key": "{.credential.token_expiry}",
			"token-key":  "{.credential.access_token}",
		},
	}

	config.Clusters["e2e-test-cluster"] = cluster
	config.Contexts["e2e-test-cluster-user"] = context
	config.AuthInfos["e2e-test-cluster-user"] = authInfo

	config.CurrentContext = "e2e-test-cluster-user"

	return config
}
