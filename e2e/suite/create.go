package suite

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"

	esV1 "github.com/matt-tyler/elasticsearch-operator/pkg/apis/es/v1"
	clusterclientset "github.com/matt-tyler/elasticsearch-operator/pkg/client/clientset/versioned"
	esclient "github.com/matt-tyler/elasticsearch-operator/pkg/client/clientset/versioned/typed/es/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//	. "github.com/onsi/gomega"
)

var _ = Describe("#Create: Creating a cluster", func() {
	var kubeInformerFactory kubeinformers.SharedInformerFactory
	var serviceLister corelisters.ServiceLister
	var ctx context.Context
	var cancel context.CancelFunc

	BeforeEach(func() {
		resyncPeriod := 0 * time.Second
		kubeclientset := kubernetes.NewForConfigOrDie(config)
		kubeInformerFactory = kubeinformers.NewSharedInformerFactory(kubeclientset, resyncPeriod)

		serviceInformer := kubeInformerFactory.Core().V1().Services()

		serviceLister = serviceInformer.Lister()

		ctx, cancel = context.WithCancel(context.Background())
		go kubeInformerFactory.Start(ctx.Done())

		if !cache.WaitForCacheSync(ctx.Done(), serviceInformer.Informer().HasSynced) {
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
		})

		AfterEach(func() {
			// delete the cluster CRD
			deletePolicy := metav1.DeletePropagationForeground
			err := clusterInterface.Delete(cluster.Name, &metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("then it should create a new cluster resource", func() {
			// watch with timeout I guess?
			Fail("test is unimplemented")
		})

		FIt("then it should create a new headless service", func() {
			timeout := time.After(time.Second * 10)
			for {
				select {
				case <-timeout:
					Fail("Headless service was not created")
				default:
					service, err := serviceLister.Services(cluster.Namespace).Get(cluster.Name + "-master-service")
					if err != nil {
						if !errors.IsNotFound(err) {
							Fail(err.Error())
						}
						continue
					}
					Expect(service.Spec.ClusterIP).To(Equal("None"))
				}
			}
		})

		It("then it should create a new replica set", func() {
			Fail("test is unimplemented")
		})
	})
})
