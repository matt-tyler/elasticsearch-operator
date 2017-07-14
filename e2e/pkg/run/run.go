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

package run

import (
	"context"
	"fmt"
	"github.com/matt-tyler/elasticsearch-operator/e2e/pkg/gke"
)

type Config struct {
	Build   bool   `mapstructure:"build"`
	Up      bool   `mapstructure:"up"`
	Down    bool   `mapstructure:"down"`
	Test    bool   `mapstructure:"test"`
	Project string `mapstructure:"project"`
	Zone    string `mapstructure:"zone"`
}

func Run(config Config, args []string) error {
	clusterId := "e2e-test-cluster"
	ctx := context.Background()
	client := gke.GkeClient{}

	if config.Up || config.Down {
		if err := gke.NewGkeClient(&client, ctx, config.Project, config.Zone); err != nil {
			return err
		}
		fmt.Println("Created GKE client")
	}

	// spin the cluster up
	if config.Up {
		fmt.Printf("Creating cluster: %v\n", clusterId)
		op, err := client.CreateCluster(clusterId)
		if err != nil {
			return err
		}
		client.Done(op)
		fmt.Printf("Cluster %v created\n", clusterId)
	}

	if config.Build {
		// build the e2e test binary
	}

	if config.Test {
		// run the tests
	}

	// spin the cluster down
	if config.Down {
		fmt.Printf("Deleting Cluster: %v\n", clusterId)
		op, err := client.DeleteCluster(clusterId)
		if err != nil {
			return err
		}

		client.Done(op)
		fmt.Printf("Cluster %v deleted\n", clusterId)
	}

	return nil
}
