package suite

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	//	. "github.com/onsi/gomega"
)

var _ = Describe("#Create: Creating a cluster", func() {

	clusterName := "es-test-cluster"
	var (
		ctx      context.Context
		cancel   context.CancelFunc
		informer cache.SharedIndexInformer
	)

	BeforeEach(func() {
		clientset := kubernetes.NewForConfigOrDie(CopyConfig(config))
		ctx, cancel = context.WithCancel(context.Background())

		indexers := map[string]cache.IndexFunc{
			"type": func(obj interface{}) ([]string, error) {
				accessor := meta.NewAccessor()
				resourceType, err := accessor.Kind(obj.(runtime.Object))
				if err != nil {
					return []string{""}, fmt.Errorf("Error accessing type of obj: %v", err)
				}
				return []string{resourceType}, nil
			},
		}

		informer = cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					return clientset.CoreV1().Services("").List(metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(map[string]string{
							"app":     "es-cluster",
							"cluster": clusterName,
						}).String(),
					})
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					return clientset.CoreV1().Services("").Watch(metav1.ListOptions{
						LabelSelector: labels.SelectorFromSet(map[string]string{
							"app":     "es-cluster",
							"cluster": clusterName,
						}).String(),
					})
				},
			},
			&core.Service{},
			0,
			indexers,
		)

		go informer.Run(ctx.Done())

		if !cache.WaitForCacheSync(
			ctx.Done(),
			informer.HasSynced,
		) {
			cancel()
			Fail("Failed waiting for cache sync")
		}
	})

	AfterEach(func() {
		cancel()
	})

	Context("Given a valid cluster definition", func() {

		JustBeforeEach(func() {
			// create the cluster CRD
		})

		AfterEach(func() {
			// delete the cluster CRD
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
					objs, err := informer.GetIndexer().ByIndex("type", "service")
					if err != nil {
						Fail(err.Error())
					}

					for _, obj := range objs {
						if service, ok := obj.(*core.Service); ok && service.Name == "" {
							Expect(service.Spec.ClusterIP).To(Equal("None"))
						}
					}
				}
			}
		})

		It("then it should create a new replica set", func() {
			Fail("test is unimplemented")
		})
	})
})
