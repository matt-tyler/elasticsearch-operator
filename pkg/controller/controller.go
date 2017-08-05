package controller

import (
	"context"
	"github.com/matt-tyler/elasticsearch-operator/pkg/log"
	"github.com/matt-tyler/elasticsearch-operator/pkg/spec"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type Controller struct {
	client *rest.RESTClient
	scheme *runtime.Scheme
}

func NewController(client *rest.RESTClient, scheme *runtime.Scheme) *Controller {
	return &Controller{client, scheme}
}

func (c *Controller) watch(ctx context.Context) (cache.Controller, error) {
	source := cache.NewListWatchFromClient(
		c.client,
		spec.ResourcePlural,
		v1.NamespaceAll,
		fields.Everything())

	_, controller := cache.NewInformer(
		source,
		&spec.Cluster{},
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.onAdd,
			UpdateFunc: c.onUpdate,
			DeleteFunc: c.onDelete,
		})

	go controller.Run(ctx.Done())
	return controller, nil
}

func (c *Controller) Run(ctx context.Context) error {
	logger := log.NewLogger()
	logger.Infof("Begin watching resources")

	_, err := c.watch(ctx)
	if err != nil {
		logger.Errorf("Failed to register watch for cluster resource: %v", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (c *Controller) onAdd(obj interface{}) {
	logger := log.NewLogger()
	cluster := obj.(spec.Cluster)

	logger.Debugf("Adding cluster: %v", cluster)

	copy := cluster.DeepCopy()
	if copy == nil {
		logger.Errorf("Failed creating a deep copy of cluster: %v", cluster)
		return
	}

	copy.Status = spec.ClusterStatus{
		State: spec.ClusterStateCreated,
	}

	err := c.client.Put().
		Name(cluster.ObjectMeta.Name).
		Namespace(cluster.ObjectMeta.Namespace).
		Resource(spec.ResourcePlural).
		Body(copy).
		Do().
		Error()

	if err != nil {
		logger.Errorf("Failed to update status: %v", err)
		return
	}

	logger.Infof("Added cluster: %v", cluster)
}

func (c *Controller) onUpdate(oldObj, newObj interface{}) {
	logger := log.NewLogger()
	oldCluster := oldObj.(spec.Cluster)
	newCluster := newObj.(spec.Cluster)
	logger.Infof("Updated: %v to %v", oldCluster, newCluster)
}

func (c *Controller) onDelete(obj interface{}) {
	logger := log.NewLogger()
	cluster := obj.(spec.Cluster)
	logger.Infof("Deleted: %v", cluster)
}
