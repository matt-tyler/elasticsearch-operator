package suite

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	esV1 "github.com/matt-tyler/elasticsearch-operator/pkg/apis/es/v1"
	clusterclientset "github.com/matt-tyler/elasticsearch-operator/pkg/client/clientset/versioned"
	esclient "github.com/matt-tyler/elasticsearch-operator/pkg/client/clientset/versioned/typed/es/v1"
	informers "github.com/matt-tyler/elasticsearch-operator/pkg/client/informers/externalversions"
	listers "github.com/matt-tyler/elasticsearch-operator/pkg/client/listers/es/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// eventQueue is for creating simple queues that indicate "something"
// happened, and it might be a good idea to recheck the state
func eventQueue(c chan<- struct{}) cache.ResourceEventHandler {
	f := func(obj interface{}) {
		c <- struct{}{}
	}
	return &cache.ResourceEventHandlerFuncs{
		AddFunc:    f,
		DeleteFunc: f,
		UpdateFunc: func(old interface{}, new interface{}) {
			f(new)
		},
	}
}

func something(handler func() (bool, error), events <-chan struct{}, timeout time.Duration) {
	t := time.After(timeout)
	for {
		select {
		case <-t:
			Fail("Timed out")
		case <-events:
			stop, err := handler()
			if err != nil {
				Fail(err.Error())
			}

			if stop {
				return
			}
		}
	}
}

var _ = Describe("#Create: Creating a cluster", func() {
	var events chan struct{}
	var serviceLister corelisters.ServiceLister
	var clusterLister listers.ClusterLister
	var ctx context.Context
	var cancel context.CancelFunc
	var label string
	var namespace string

	BeforeEach(func() {
		events = make(chan struct{}, 10)
		handlers := eventQueue(events)
		label = "test"
		namespace = "default"

		resyncPeriod := 0 * time.Second

		listOptions := func(options *metav1.ListOptions) {
			options.LabelSelector = labels.Set(map[string]string{
				"test": label,
			}).AsSelector().String()
		}

		kubeclientset := kubernetes.NewForConfigOrDie(config)
		kubeInformerFactory := kubeinformers.NewFilteredSharedInformerFactory(kubeclientset, resyncPeriod, namespace, listOptions)

		esclientset := clusterclientset.NewForConfigOrDie(config)
		esInformerFactory := informers.NewFilteredSharedInformerFactory(esclientset, resyncPeriod, namespace, listOptions)

		serviceInformer := kubeInformerFactory.Core().V1().Services()
		serviceLister = serviceInformer.Lister()
		serviceInformer.Informer().AddEventHandler(handlers)

		clusterInformer := esInformerFactory.Es().V1().Clusters()
		clusterLister = clusterInformer.Lister()
		clusterInformer.Informer().AddEventHandler(handlers)

		ctx, cancel = context.WithCancel(context.Background())

		go kubeInformerFactory.Start(ctx.Done())
		go esInformerFactory.Start(ctx.Done())

		if !cache.WaitForCacheSync(ctx.Done(),
			serviceInformer.Informer().HasSynced,
			clusterInformer.Informer().HasSynced) {
			Fail("Timed out waiting for cache sync")
		}
	})

	AfterEach(func() {
		cancel()
	})

	Context("Given a valid cluster definition", func() {
		var clusterInterface esclient.ClusterInterface
		var cluster *esV1.Cluster

		BeforeEach(func() {
			clusterInterface = clusterclientset.NewForConfigOrDie(config).
				EsV1().Clusters("default")

			cluster = &esV1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "example-cluster",
					Namespace: "default",
					Labels: map[string]string{
						"test": label,
					},
				},
				Spec: esV1.ClusterSpec{
					Name: "example-cluster",
					Size: 1,
				},
			}
		})

		JustBeforeEach(func() {
			// create the cluster CRD
			var err error
			cluster, err = clusterInterface.Create(cluster)
			Expect(err).NotTo(HaveOccurred())

			check := func() (bool, error) {
				_, err := clusterLister.Clusters(cluster.Namespace).Get(cluster.Name)
				if err == nil {
					return true, nil
				}

				if errors.IsNotFound(err) {
					return false, nil
				}

				return true, err
			}

			something(check, events, time.Second*10)
		})

		AfterEach(func() {
			// delete the cluster CRD
			deletePolicy := metav1.DeletePropagationForeground
			err := clusterInterface.Delete(cluster.Name, &metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			})
			Expect(err).NotTo(HaveOccurred())

			check := func() (bool, error) {
				_, err := clusterLister.Clusters(cluster.Namespace).Get(cluster.Name)
				if err == nil {
					return false, nil
				}

				if errors.IsNotFound(err) {
					return true, nil
				}

				return false, err
			}
			// It can take awhile for the garbage collector
			// to pick changes up = flaky test
			// Requires doing some investigation into GC behaviour
			// At worst, requires explicitly deleting some dependencies
			something(check, events, time.Second*30)
		})

		It("then it should create a new cluster resource", func() {
			// watch with timeout I guess?
			Fail("test is unimplemented")
		})

		FIt("then it should create a new headless service", func() {
			check := func() (bool, error) {
				service, err := serviceLister.Services(cluster.Namespace).Get(cluster.Name + "-master-service")
				if err != nil {
					if errors.IsNotFound(err) {
						// keep retrying
						return false, nil
					}
					// some other error, so terminate
					return true, err
				}

				Expect(service.Spec.ClusterIP).To(Equal("None"))
				return true, nil
			}

			something(check, events, time.Second*10)
		})

		It("then it should create a new replica set", func() {
			Fail("test is unimplemented")
		})
	})
})
