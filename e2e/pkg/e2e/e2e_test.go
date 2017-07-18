package e2e

import (
	_ "github.com/matt-tyler/elasticsearch-operator/e2e/pkg/e2e/example"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}

func RunE2ETests(t *testing.T) {
	RegisterFailHandler(Fail)

	r := make([]Reporter, 0)

	RunSpecsWithDefaultAndCustomReporters(t, "Elasticsearch Operator E2E Suite", r)
}
