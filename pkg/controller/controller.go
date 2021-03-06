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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"

	clusterscheme "github.com/matt-tyler/elasticsearch-operator/pkg/client/clientset/versioned/scheme"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	appslisters "k8s.io/client-go/listers/apps/v1beta2"
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

	clustersSynced    cache.InformerSynced
	servicesSynced    cache.InformerSynced
	deploymentsSynced cache.InformerSynced

	clusterLister    listers.ClusterLister
	serviceLister    corelisters.ServiceLister
	deploymentLister appslisters.DeploymentLister

	queue workqueue.RateLimitingInterface

	recorder record.EventRecorder
}

func NewController(config *rest.Config) *Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	kubeclientset := kubernetes.NewForConfigOrDie(config)
	esclientset := clientset.NewForConfigOrDie(config)

	resyncPeriod := 0 * time.Second

	listOptions := func(options *metav1.ListOptions) {
		options.LabelSelector = labels.Set(map[string]string{
			"operator": "elasticsearch-operator",
		}).AsSelector().String()
	}

	kubeInformerFactory := kubeinformers.NewFilteredSharedInformerFactory(kubeclientset, resyncPeriod, "", listOptions)
	esInformerFactory := informers.NewSharedInformerFactory(esclientset, resyncPeriod)

	clusterInformer := esInformerFactory.Es().V1().Clusters()
	serviceInformer := kubeInformerFactory.Core().V1().Services()
	deploymentInformer := kubeInformerFactory.Apps().V1beta2().Deployments()

	logger := log.NewLogger()

	clusterscheme.AddToScheme(scheme.Scheme)
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(logger.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		logger,
		kubeclientset,
		esclientset,
		kubeInformerFactory,
		esInformerFactory,
		clusterInformer.Informer().HasSynced,
		serviceInformer.Informer().HasSynced,
		deploymentInformer.Informer().HasSynced,
		clusterInformer.Lister(),
		serviceInformer.Lister(),
		deploymentInformer.Lister(),
		queue,
		recorder,
	}

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

	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			newSvc := newObj.(*corev1.Service)
			oldSvc := oldObj.(*corev1.Service)
			if newSvc.ResourceVersion == oldSvc.ResourceVersion {
				return
			}
			controller.handleObject(newObj)
		},
		DeleteFunc: controller.handleObject,
	})

	ctx := context.Background()

	go kubeInformerFactory.Start(ctx.Done())
	go esInformerFactory.Start(ctx.Done())

	return controller
}

func (c *Controller) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
		}
		c.Infof("Recovered deleted object '%s' from tombstone", object.GetName)
	}
	c.Infof("Processing object: %s", object.GetName())
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		if ownerRef.Kind != "Cluster" {
			return
		}

		cluster, err := c.clusterLister.Clusters(object.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			c.Infof("Ignoring orphaned object '%s' of cluster '%s'", object.GetSelfLink(), ownerRef.Name)
			return
		}

		var key string

		if key, err = cache.MetaNamespaceKeyFunc(cluster); err != nil {
			runtime.HandleError(err)
			return
		}
		c.queue.AddRateLimited(key)
	}
}

func (c *Controller) runWorker() {
	c.Infof("Processing items")
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

	c.Infof("Creating master node deployment...")
	masterDeploymentName := fmt.Sprintf("%v-master-deployment", cluster.Name)
	masterDeployment, err := c.deploymentLister.Deployments(cluster.Namespace).Get(masterDeploymentName)
	if errors.IsNotFound(err) {
		masterDeployment, err = c.kubeclientset.AppsV1beta2().Deployments(cluster.Namespace).Create(newMasterDeployment(cluster, masterServiceName))
	}

	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(masterDeployment, cluster) {
		msg := fmt.Sprintf(MessageResourceExists, masterDeployment.Name)
		c.recorder.Event(cluster, corev1.EventTypeWarning, ErrResourceExists, msg)
		return fmt.Errorf(msg)
	}

	msg := fmt.Sprintf(MessageResourceSynced, cluster.Name)
	c.recorder.Event(cluster, corev1.EventTypeNormal, SuccessSynced, msg)

	return nil
}

func (c *Controller) Run(ctx context.Context) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	c.Infof("Starting Controller...")

	if !cache.WaitForCacheSync(ctx.Done(), c.clustersSynced, c.servicesSynced) {
		utilruntime.HandleError(fmt.Errorf("Timed out waiting for cache to sync"))
		return
	}

	c.Infof("Controller started")

	wait.Until(c.runWorker, time.Second, ctx.Done())
}
