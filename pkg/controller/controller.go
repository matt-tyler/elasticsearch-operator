package controller

import (
	"context"
	"fmt"
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
	fmt.Println("Begin watching resources")

	_, err := c.watch(ctx)
	if err != nil {
		fmt.Printf("Failed to register watch for Example resource: %v\n", err)
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func (c *Controller) onAdd(obj interface{}) {

}

func (c *Controller) onUpdate(oldObj, newObj interface{}) {

}

func (c *Controller) onDelete(obj interface{}) {

}
