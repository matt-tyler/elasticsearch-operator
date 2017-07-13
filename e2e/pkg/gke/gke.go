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
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	"net/http"
	"time"
)

type GkeClient struct {
	ProjectId string
	Zone      string
	client    *http.Client
}

func NewGkeClient(gkeClient *GkeClient, ctx context.Context, projectId, zone string) error {
	client, err := google.DefaultClient(ctx, container.CloudPlatformScope)
	if err != nil {
		return err
	}
	gkeClient = &GkeClient{projectId, zone, client}
	return nil
}

func (c *GkeClient) DeleteCluster(clusterId string) (string, error) {
	service, err := container.New(c.client)
	if err != nil {
		return "", err
	}

	projectsZonesClustersService := container.NewProjectsZonesClustersService(service)
	projectZonesClustersDeleteCall := projectsZonesClustersService.Delete(c.ProjectId, c.Zone, clusterId)

	op, err := projectZonesClustersDeleteCall.Do()
	if err != nil {
		return "", err
	}

	return op.Name, nil
}

func GetClusterStatus(client *http.Client, clusterId, projectId, zone string) (*string, error) {
	service, err := container.New(client)
	if err != nil {
		return nil, err
	}

	projectsZonesClustersService := container.NewProjectsZonesClustersService(service)
	projectZonesClustersGetCall := projectsZonesClustersService.Get(projectId, zone, clusterId)
	projectZonesClustersGetCall.Fields("status")

	cluster, err := projectZonesClustersGetCall.Do()
	if err != nil {
		return nil, err
	}

	return &(cluster.Status), nil
}

func ListClusters(client *http.Client, projectId, zone string) (*container.ListClustersResponse, error) {
	service, err := container.New(client)
	if err != nil {
		return nil, err
	}

	projectsZonesClustersService := container.NewProjectsZonesClustersService(service)
	projectsZonesClustersListCall := projectsZonesClustersService.List(projectId, zone)

	projectsZonesClustersListCall.Fields("clusters(name,status)")

	listClustersResponse, err := projectsZonesClustersListCall.Do()
	if err != nil {
		return nil, err
	}

	return listClustersResponse, nil
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
			InitialClusterVersion: "1.6.4",
			InitialNodeCount:      3,
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

	createCall := projectsZonesClustersService.Create(c.ProjectId, c.Zone, createClusterRequest)
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
	projectsZonesOperationsGetCall := projectsZonesOperationsService.Get(c.ProjectId, c.Zone, operationId)
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
