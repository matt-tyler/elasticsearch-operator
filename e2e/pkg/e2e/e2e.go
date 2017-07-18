package e2e

import (
	_ "github.com/matt-tyler/elasticsearch-operator/e2e/pkg/e2e/example"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestExample(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Example Suite")
}
