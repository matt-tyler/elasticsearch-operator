package main

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	"net/http"
	"os"
)

func main() {
	projectId := os.Getenv("PROJECT_ID")
	zone := os.Getenv("REGION")

	ctx := context.Background()
	client, err := google.DefaultClient(ctx, container.CloudPlatformScope)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err := Create(client, projectId, zone); err != nil {
		fmt.Println(err.Error())
	}

	if status, err := GetClusterStatus(client, "e2e-test-cluster", projectId, zone); err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println(*status)
	}
}

func DeleteCluster(client *http.Client, clusterId, projectId, zone string) error {
	service, err := container.New(client)
	if err != nil {
		return nil, err
	}

	projectsZonesClustersService := container.NewProjectsZonesClustersService(service)
	projectZonesClustersDeleteCall := projectsZonesClustersService.Delete(projectId, zone, clusterId)

	return projectZonesClustersDeleteCall.Do()
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

func Create(client *http.Client, projectId, zone string) error {
	service, err := container.New(client)
	if err != nil {
		return err
	}

	projectsZonesClustersService := container.NewProjectsZonesClustersService(service)

	createClusterRequest := &container.CreateClusterRequest{
		Cluster: &container.Cluster{
			Name:                  "e2e-test-cluster",
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

	createCall := projectsZonesClustersService.Create(projectId, zone, createClusterRequest)
	op, err := createCall.Do()
	if err != nil {
		return err
	}

	fmt.Println(op.Status)

	if op.Status != "RUNNING" {
		return errors.New("Failed to create cluster")
	}

	return nil
}
