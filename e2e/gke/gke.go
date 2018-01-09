// Copyright © 2017 Matt Tyler <me@matthewtyler.io>
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

package gke

import (
	"context"
	"encoding/base64"
	"net/http"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	rest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type GkeClient struct {
	Project string
	Zone    string
	client  *http.Client
}

func NewGkeClient(gkeClient *GkeClient, ctx context.Context, project, zone string) error {
	client, err := google.DefaultClient(ctx, container.CloudPlatformScope)
	if err != nil {
		return err
	}
	*gkeClient = GkeClient{project, zone, client}
	return nil
}

func (c *GkeClient) DeleteCluster(clusterId string) (string, error) {
	service, err := container.New(c.client)
	if err != nil {
		return "", err
	}

	projectsZonesClustersService := container.NewProjectsZonesClustersService(service)
	projectZonesClustersDeleteCall := projectsZonesClustersService.Delete(c.Project, c.Zone, clusterId)

	op, err := projectZonesClustersDeleteCall.Do()
	if err != nil {
		return "", err
	}

	return op.Name, nil
}

func (c *Cluster) Config() (*rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags(c.Endpoint, "")
	if err != nil {
		return nil, err
	}

	config.AuthProvider = &clientcmdapi.AuthProviderConfig{
		Name: "gcp",
	}

	cacert, _ := base64.StdEncoding.DecodeString(c.Auth.ClusterCaCertificate)

	config.TLSClientConfig = rest.TLSClientConfig{
		CAData: cacert,
	}

	return config, nil
}

type Cluster struct {
	Auth     *container.MasterAuth
	Status   string
	Endpoint string
}

func (c *GkeClient) GetCluster(clusterId string) (*Cluster, error) {
	service, err := container.New(c.client)
	if err != nil {
		return nil, err
	}

	projectsZonesClustersService := container.NewProjectsZonesClustersService(service)
	projectZonesClustersGetCall := projectsZonesClustersService.Get(c.Project, c.Zone, clusterId)
	projectZonesClustersGetCall.Fields("status,endpoint,masterAuth")

	cluster, err := projectZonesClustersGetCall.Do()
	if err != nil {
		return nil, err
	}

	return &Cluster{cluster.MasterAuth, cluster.Status, cluster.Endpoint}, nil
}

func (c *GkeClient) CreateCluster(clusterId string) (string, error) {
	service, err := container.New(c.client)
	if err != nil {
		return "", err
	}

	projectsZonesClustersService := container.NewProjectsZonesClustersService(service)

	createClusterRequest := &container.CreateClusterRequest{
		Cluster: &container.Cluster{
			Name:                  clusterId,
			Description:           "A cluster for e2e testing of elasticsearch-operator",
			InitialClusterVersion: "1.8.5-gke.0",
			InitialNodeCount:      3,
			EnableKubernetesAlpha: true,
			NodeConfig: &container.NodeConfig{
				DiskSizeGb:  40,
				ImageType:   "COS",
				MachineType: "f1-micro",
				OauthScopes: []string{
					"https://www.googleapis.com/auth/compute",
					"https://www.googleapis.com/auth/devstorage.read_only",
					"https://www.googleapis.com/auth/logging.write",
					"https://www.googleapis.com/auth/monitoring.write",
					"https://www.googleapis.com/auth/servicecontrol",
					"https://www.googleapis.com/auth/service.management.readonly",
					"https://www.googleapis.com/auth/trace.append",
				},
			},
		},
	}

	createCall := projectsZonesClustersService.Create(c.Project, c.Zone, createClusterRequest)
	op, err := createCall.Do()
	if err != nil {
		return "", err
	}

	return op.Name, nil
}

func (c *GkeClient) Done(operationId string) error {
	service, err := container.New(c.client)
	if err != nil {
		return err
	}

	projectsZonesOperationsService := container.NewProjectsZonesOperationsService(service)
	projectsZonesOperationsGetCall := projectsZonesOperationsService.Get(c.Project, c.Zone, operationId)
	projectsZonesOperationsGetCall.Fields("status")

	DoGetCall := func() (*container.Operation, error) {
		return projectsZonesOperationsGetCall.Do()
	}

	for op, err := DoGetCall(); op.Status != "DONE"; op, err = DoGetCall() {
		if err != nil {
			return err
		}

		time.Sleep(10 * time.Second)
	}

	return nil
}
