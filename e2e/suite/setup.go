package suite

import (
	"bytes"
	"encoding/json"
	"html/template"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/api/apps/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	appsv1beta1 "k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

var deployment *v1beta1.Deployment
var events <-chan watch.Event

type Params struct {
	Image string
}

var deploymentTemplate = `
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

// Setup registers the custom resource definition/s
func Setup(config *rest.Config, image string) {
	BeforeSuite(func() {
		apiextensionsclientset := apiextensionsclient.NewForConfigOrDie(config)

		w, err := apiextensionsclientset.ApiextensionsV1beta1().
			CustomResourceDefinitions().Watch(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		events = w.ResultChan()

		clientset := appsv1beta1.NewForConfigOrDie(config)

		deploymentClient := clientset.Deployments(metav1.NamespaceDefault)

		buf := &bytes.Buffer{}
		p := &Params{
			image,
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
		case event := <-events:
			_ = event.Object.(*apiextensionsv1beta1.CustomResourceDefinition)
			Expect(event.Type).To(Equal(watch.Added))
		case <-time.After(time.Second * 10):
			Fail("Creating custom resource definition exceeded time out.")
		}
	})

	AfterSuite(func() {
		clientset := appsv1beta1.NewForConfigOrDie(config)

		deploymentClient := clientset.Deployments(metav1.NamespaceDefault)

		deletePolicy := metav1.DeletePropagationForeground
		err := deploymentClient.Delete(deployment.Name, &metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		})
		Expect(err).NotTo(HaveOccurred())

		timeout := time.After(time.Second * 10)

		for {
			select {
			case <-timeout:
				Fail("Deleting custom resource definition exceeded time out.")
			default:
				event := <-events
				if event.Type == watch.Deleted {
					return
				}
			}
		}
	})
}
