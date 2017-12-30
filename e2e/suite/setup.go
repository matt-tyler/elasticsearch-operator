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

	"k8s.io/api/apps/v1beta2"
	coreV1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	appsv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

var config *rest.Config

type Params struct {
	Image string
}

var serviceAccountTemplate = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: e2e-service-account
`

var clusterRoleTemplate = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: e2e-test-role
rules:
`

var clusterRoleCRDTemplate = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: e2e-test-role-crd
rules:
- apiGroups: ["apiextensions.k8s.io"]
  resources: ["customresourcedefinitions"]
  verbs: ["*"]
- apiGroups: ["", "apps"]
  resources: ["deployments", "services" ]
  verbs: ["*"]
- apiGroups: ["es.matt-tyler.github.com"]
  resources: ["clusters"]
  verbs: ["*"]
`

var clusterRoleBindingTemplate = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: e2e-test-role-cluster-binding
roleRef:
  kind: ClusterRole
  name: e2e-test-role-crd
  apiGroup: rbac.authorization.k8s.io
`

var deploymentTemplate = `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
  name: elasticsearch-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: elasticsearch-operator  
  template:
    metadata:
      labels:
        app: elasticsearch-operator
    spec:
      serviceAccountName: e2e-service-account
      containers:
      - name: elasticsearch-operator
        image: {{.Image}}
`

func createClusterRoles(clientset kubernetes.Interface) ([]*rbacV1.ClusterRole, error) {
	clusterRole := &rbacV1.ClusterRole{}
	roles := []*rbacV1.ClusterRole{}

	clusterRoleJSON, err := yaml.ToJSON([]byte(clusterRoleTemplate))
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(clusterRoleJSON, &clusterRole); err != nil {
		return nil, err
	}

	role, err := clientset.RbacV1().ClusterRoles().Create(clusterRole)
	if err != nil {
		return nil, err
	}
	roles = append(roles, role)

	clusterRoleJSON, err = yaml.ToJSON([]byte(clusterRoleCRDTemplate))
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(clusterRoleJSON, &clusterRole); err != nil {
		return nil, err
	}

	role, err = clientset.RbacV1().ClusterRoles().Create(clusterRole)
	if err != nil {
		return nil, err
	}

	return append(roles, role), nil
}

func deleteClusterRole(clusterRole *rbacV1.ClusterRole, clientset kubernetes.Interface) error {
	return clientset.RbacV1().ClusterRoles().Delete(clusterRole.Name, nil)
}

func createServiceAccount(namespace string, clientset kubernetes.Interface) (*coreV1.ServiceAccount, error) {
	serviceAccount := &coreV1.ServiceAccount{}

	buf := &bytes.Buffer{}
	p := struct {
		Namespace string
	}{namespace}

	tmpl := template.Must(template.New("").Parse(serviceAccountTemplate))
	err := tmpl.Execute(buf, p)
	if err != nil {
		return nil, err
	}

	serviceAccountJSON, err := yaml.ToJSON(buf.Bytes())
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(serviceAccountJSON, &serviceAccount); err != nil {
		return nil, err
	}

	return clientset.CoreV1().ServiceAccounts(namespace).Create(serviceAccount)
}

func deleteServiceAccount(serviceAccount *coreV1.ServiceAccount, clientset kubernetes.Interface) error {
	return clientset.CoreV1().ServiceAccounts(serviceAccount.Namespace).Delete(serviceAccount.Name, nil)
}

func createClusterRoleBinding(serviceAccount *coreV1.ServiceAccount, clientset kubernetes.Interface) (*rbacV1.ClusterRoleBinding, error) {
	clusterRoleBinding := &rbacV1.ClusterRoleBinding{}

	clusterRoleBindingJSON, err := yaml.ToJSON([]byte(clusterRoleBindingTemplate))
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(clusterRoleBindingJSON, &clusterRoleBinding); err != nil {
		return nil, err
	}

	clusterRoleBinding.Subjects = []rbacV1.Subject{{
		Kind:      "ServiceAccount",
		Name:      serviceAccount.Name,
		Namespace: serviceAccount.Namespace,
		APIGroup:  "",
	}}

	return clientset.RbacV1().ClusterRoleBindings().Create(clusterRoleBinding)
}

func deleteClusterRoleBinding(clusterRoleBinding *rbacV1.ClusterRoleBinding, clientset kubernetes.Interface) error {
	return clientset.RbacV1().ClusterRoleBindings().Delete(clusterRoleBinding.Name, nil)
}

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

	ctx, cancel := context.WithCancel(context.Background())

	go informer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		cancel()
		return errors.New("Failed waiting for cache sync")
	}

	var deployment *v1beta2.Deployment

	clusterRoles := []*rbacV1.ClusterRole{}
	var serviceAccount *coreV1.ServiceAccount
	var clusterRoleBinding *rbacV1.ClusterRoleBinding

	BeforeSuite(func() {
		var err error

		k8s := kubernetes.NewForConfigOrDie(CopyConfig(config))
		clusterRoles, err = createClusterRoles(k8s)
		Expect(err).NotTo(HaveOccurred())

		serviceAccount, err = createServiceAccount("default", k8s)
		Expect(err).NotTo(HaveOccurred())

		clusterRoleBinding, err = createClusterRoleBinding(serviceAccount, k8s)
		Expect(err).NotTo(HaveOccurred())

		clientset := appsv1beta2.NewForConfigOrDie(CopyConfig(config))

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
		clientset := appsv1beta2.NewForConfigOrDie(CopyConfig(config))

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
					k8s := kubernetes.NewForConfigOrDie(CopyConfig(config))

					if clusterRoleBinding != nil {
						_ = deleteClusterRoleBinding(clusterRoleBinding, k8s)
					}

					if serviceAccount != nil {
						_ = deleteServiceAccount(serviceAccount, k8s)
					}

					for _, role := range clusterRoles {
						_ = deleteClusterRole(role, k8s)
					}

					cancel()
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
