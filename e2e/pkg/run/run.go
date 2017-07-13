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
	Build     bool   `mapstructure:"build"`
	Up        bool   `mapstructure:"up"`
	Down      bool   `mapstructure:"down"`
	Test      bool   `mapstructure:"test"`
	ProjectId string `mapstructure:"projectid"`
	Zone      string `mapstructure:"zone"`
}

func Run(config Config, args []string) error {
	clusterId := "e2e-test-cluster"
	ctx := context.Background()
	client := gke.GkeClient{}

	if config.Up || config.Down {
		if err := gke.NewGkeClient(&client, ctx, config.ProjectId, config.Zone); err != nil {
			return err
		}
	}

	// spin the cluster up
	if config.Up {
		fmt.Println("Creating cluster: %v", clusterId)
		op, err := client.CreateCluster(clusterId)
		if err != nil {
			return err
		}

		client.Done(op)
		fmt.Println("Cluster %v created", clusterId)
	}

	if config.Build {
		// build the e2e test binary
	}

	if config.Test {
		// run the tests
	}

	// spin the cluster down
	if config.Down {
		fmt.Println("Deleting Cluster: %v", clusterId)
		op, err := client.DeleteCluster(clusterId)
		if err != nil {
			return err
		}

		client.Done(op)
		fmt.Println("Cluster %v deleted", clusterId)
	}

	return nil
}
