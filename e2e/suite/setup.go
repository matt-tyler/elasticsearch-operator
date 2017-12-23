package suite

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/api/apps/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	appsv1beta1 "k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var config *rest.Config

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
func Setup(c *rest.Config, image string) error {

	config = c

	// TODO: Use CopyConfig when bumping client-go to >= 4.0

	apiextensionsclientset := apiextensionsclient.NewForConfigOrDie(CopyConfig(config))

	queue := workqueue.New()

	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return apiextensionsclientset.ApiextensionsV1beta1().
					CustomResourceDefinitions().List(metav1.ListOptions{})
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return apiextensionsclientset.ApiextensionsV1beta1().
					CustomResourceDefinitions().Watch(metav1.ListOptions{})
			},
		},
		&apiextensionsv1beta1.CustomResourceDefinition{},
		0,
		cache.Indexers{},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if key, err := cache.MetaNamespaceKeyFunc(obj); err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj); err == nil {
				queue.Add(key)
			}
		},
	})

	ctx := context.Background()

	go informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		return errors.New("Failed waiting for cache sync")
	}

	var deployment *v1beta1.Deployment

	BeforeSuite(func() {
		clientset := appsv1beta1.NewForConfigOrDie(CopyConfig(config))

		deploymentClient := clientset.Deployments(metav1.NamespaceDefault)

		buf := &bytes.Buffer{}
		p := &Params{
			image,
		}

		tmpl := template.Must(template.New("").Parse(deploymentTemplate))
		err := tmpl.Execute(buf, p)
		Expect(err).NotTo(HaveOccurred())

		deploymentJSON, err := yaml.ToJSON(buf.Bytes())
		Expect(err).NotTo(HaveOccurred())

		err = json.Unmarshal(deploymentJSON, &deployment)
		Expect(err).NotTo(HaveOccurred())

		deployment, err = deploymentClient.Create(deployment)
		Expect(err).NotTo(HaveOccurred())

		timeout := time.After(time.Second * 10)

		for {
			select {
			case <-timeout:
				Fail("Creating custom resource definition exceeded timeout")
			default:
				key, _ := queue.Get()
				defer queue.Done(key)

				_, exists, err := informer.GetIndexer().GetByKey(key.(string))
				if err != nil {
					Fail(err.Error())
				}

				if exists {
					return
				}
			}
		}
	})

	AfterSuite(func() {
		clientset := appsv1beta1.NewForConfigOrDie(CopyConfig(config))

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
				key, _ := queue.Get()
				defer queue.Done(key)

				_, exists, err := informer.GetIndexer().GetByKey(key.(string))
				if err != nil {
					Fail(err.Error())
				}

				if !exists {
					queue.ShutDown()
					return
				}
			}
		}
	})

	return nil
}

func CopyConfig(config *rest.Config) *rest.Config {
	return &rest.Config{
		Host:          config.Host,
		APIPath:       config.APIPath,
		Prefix:        config.Prefix,
		ContentConfig: config.ContentConfig,
		Username:      config.Username,
		Password:      config.Password,
		BearerToken:   config.BearerToken,
		Impersonate: rest.ImpersonationConfig{
			Groups:   config.Impersonate.Groups,
			Extra:    config.Impersonate.Extra,
			UserName: config.Impersonate.UserName,
		},
		AuthProvider:        config.AuthProvider,
		AuthConfigPersister: config.AuthConfigPersister,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure:   config.TLSClientConfig.Insecure,
			ServerName: config.TLSClientConfig.ServerName,
			CertFile:   config.TLSClientConfig.CertFile,
			KeyFile:    config.TLSClientConfig.KeyFile,
			CAFile:     config.TLSClientConfig.CAFile,
			CertData:   config.TLSClientConfig.CertData,
			KeyData:    config.TLSClientConfig.KeyData,
			CAData:     config.TLSClientConfig.CAData,
		},
		UserAgent:     config.UserAgent,
		Transport:     config.Transport,
		WrapTransport: config.WrapTransport,
		QPS:           config.QPS,
		Burst:         config.Burst,
		RateLimiter:   config.RateLimiter,
		Timeout:       config.Timeout,
	}
}
