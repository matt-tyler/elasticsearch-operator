package controller

import (
	"context"
	"fmt"
	"time"

	clientset "github.com/matt-tyler/elasticsearch-operator/pkg/client/clientset/versioned"
	"github.com/matt-tyler/elasticsearch-operator/pkg/client/clientset/versioned/scheme"
	informers "github.com/matt-tyler/elasticsearch-operator/pkg/client/informers/externalversions"
	listers "github.com/matt-tyler/elasticsearch-operator/pkg/client/listers/es/v1"
	"github.com/matt-tyler/elasticsearch-operator/pkg/log"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"

	clusterscheme "github.com/matt-tyler/elasticsearch-operator/pkg/client/clientset/versioned/scheme"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

const maxRetries = 5
const controllerAgentName = "elasticsearch-cluster-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a cluster is synced
	SuccessSynced = "Synced"

	// ErrResourceExists is used as part of the Event 'reason' when a cluster fails
	// to sync due to a resource already existing
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceExists is the message used for events when a resource
	// fails to sync due to it already existing
	MessageResourceExists = "Resource %q already exists and is not managed by controller"

	// MessageResourceSynced is the messaged used for an Event fire when a Cluster
	// is successfully synced.
	MessageResourceSynced = "Resource %q synced successfully"
)

type Controller struct {
	log.Logger

	kubeclientset kubernetes.Interface
	esclientset   clientset.Interface

	kubeInformerFactory kubeinformers.SharedInformerFactory
	esInformerFactory   informers.SharedInformerFactory

	clustersSynced cache.InformerSynced
	servicesSynced cache.InformerSynced

	clusterLister listers.ClusterLister
	serviceLister corelisters.ServiceLister

	queue workqueue.RateLimitingInterface

	recorder record.EventRecorder
}

func NewController(config *rest.Config) *Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	kubeclientset := kubernetes.NewForConfigOrDie(config)
	esclientset := clientset.NewForConfigOrDie(config)

	resyncPeriod := 0 * time.Second

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, resyncPeriod)
	esInformerFactory := informers.NewSharedInformerFactory(esclientset, resyncPeriod)

	clusterInformer := esInformerFactory.Es().V1().Clusters()
	serviceInformer := kubeInformerFactory.Core().V1().Services()

	clusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if key, err := cache.MetaNamespaceKeyFunc(obj); err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			if key, err := cache.MetaNamespaceKeyFunc(newObj); err == nil {
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

	go kubeInformerFactory.Start(ctx.Done())
	go esInformerFactory.Start(ctx.Done())

	logger := log.NewLogger()

	clusterscheme.AddToScheme(scheme.Scheme)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(logger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	return &Controller{
		logger,
		kubeclientset,
		esclientset,
		kubeInformerFactory,
		esInformerFactory,
		clusterInformer.Informer().HasSynced,
		serviceInformer.Informer().HasSynced,
		clusterInformer.Lister(),
		serviceInformer.Lister(),
		queue,
		recorder,
	}
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
		// continue looping
	}
}

func (c *Controller) processNextItem() bool {
	obj, shutdown := c.queue.Get()

	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.queue.Done(obj)

		var key string
		var ok bool

		if key, ok = obj.(string); !ok {
			c.queue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		if err := c.sync(key); err != nil {
			return fmt.Errorf("error syncing '%s': %s", key, err.Error())
		}

		c.queue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
	}

	return true
}

func (c *Controller) sync(key string) error {
	c.Infof("Processing change to %s", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	cluster, err := c.clusterLister.Clusters(namespace).Get(name)
	if err != nil {
		if errors.IsNotFound(err) {
			runtime.HandleError(fmt.Errorf("foo '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	c.Infof("Object: %#v", cluster)

	c.Infof("create master discovery service...")
	masterServiceName := fmt.Sprintf("%v-master-service", cluster.Name)
	masterService, err := c.serviceLister.Services(cluster.Namespace).Get(masterServiceName)
	if errors.IsNotFound(err) {
		masterService, err = c.kubeclientset.CoreV1().Services(cluster.Namespace).Create(newMasterService(cluster))
	}

	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(masterService, cluster) {
		msg := fmt.Sprintf(MessageResourceExists, masterService.Name)
		c.recorder.Event(cluster, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	c.recorder.Event(cluster, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)

	return nil
}

func (c *Controller) Run(ctx context.Context) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	c.Infof("Starting Controller...")

	if !cache.WaitForCacheSync(ctx.Done(), c.clustersSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for cache to sync"))
		return
	}

	wait.Until(c.runWorker, time.Second, ctx.Done())
}
