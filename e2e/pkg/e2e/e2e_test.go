package e2e

import (
	"bytes"
	"encoding/json"
	"flag"
	_ "github.com/matt-tyler/elasticsearch-operator/e2e/pkg/e2e/example"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/apps/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	appsv1beta1 "k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"testing"
	"text/template"
	"time"
)

var Kubeconfig string
var Image string

type Params struct {
	Image string
}

func init() {
	flag.StringVar(&Kubeconfig, "kubeconfig", "", "Location of kubeconfig")
	flag.StringVar(&Image, "image", "gcr.io/schnauzer-163208/elasticsearch-operator:latest", "image under test")
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

var deployment *v1beta1.Deployment

var _ = BeforeSuite(func() {
	config, err := buildConfig(Kubeconfig)
	Expect(err).NotTo(HaveOccurred())

	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred())

	w, err := apiextensionsclientset.ApiextensionsV1beta1().
		CustomResourceDefinitions().Watch(metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())

	clientset, err := appsv1beta1.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred())

	deploymentClient := clientset.Deployments(metav1.NamespaceDefault)

	deploymentTemplate := `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: elasticsearch-operator
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: elasticsearch-operator
    spec:
      containers:
      - name: elasticsearch-operator
        image: {{.Image}}
`

	buf := &bytes.Buffer{}
	p := &Params{
		Image,
	}

	tmpl := template.Must(template.New("").Parse(deploymentTemplate))
	err = tmpl.Execute(buf, p)
	Expect(err).NotTo(HaveOccurred())

	deploymentJSON, err := yaml.ToJSON(buf.Bytes())
	Expect(err).NotTo(HaveOccurred())

	err = json.Unmarshal(deploymentJSON, &deployment)
	Expect(err).NotTo(HaveOccurred())

	deployment, err = deploymentClient.Create(deployment)
	Expect(err).NotTo(HaveOccurred())

	select {
	case event := <-w.ResultChan():
		Expect(event.Type).To(Equal(watch.Added))
		_ = event.Object.(*apiextensionsv1beta1.CustomResourceDefinition)
	case <-time.After(time.Second * 10):
		Fail("Creating custom resource definition exceeded time out.")
	}
})

var _ = AfterSuite(func() {
	config, err := buildConfig(Kubeconfig)
	Expect(err).NotTo(HaveOccurred())

	apiextensionsclientset, err := apiextensionsclient.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred())

	w, err := apiextensionsclientset.ApiextensionsV1beta1().
		CustomResourceDefinitions().Watch(metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())

	clientset, err := appsv1beta1.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred())

	deploymentClient := clientset.Deployments(metav1.NamespaceDefault)

	deletePolicy := metav1.DeletePropagationForeground
	err = deploymentClient.Delete(deployment.Name, &metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	Expect(err).NotTo(HaveOccurred())

	select {
	case event := <-w.ResultChan():
		Expect(event.Type).To(Equal(watch.Deleted))
	case <-time.After(time.Second * 5):
		Fail("Deleting custom resource definition exceeded time out.")
	}
})

func RunE2ETests(t *testing.T) {
	RegisterFailHandler(Fail)

	r := make([]Reporter, 0)

	RunSpecsWithDefaultAndCustomReporters(t, "Elasticsearch Operator E2E Suite", r)
}
