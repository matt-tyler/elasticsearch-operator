package suite

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
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

type Lister struct {
	label        string
	namespace    string
	resyncPeriod time.Duration
	events       chan<- struct{}
}

func SetResyncPeriod(d time.Duration) func(*Lister) {
	return func(lister *Lister) {
		lister.resyncPeriod = d
	}
}

func SetNamespace(namespace string) func(*Lister) {
	return func(lister *Lister) {
		lister.namespace = namespace
	}
}

func SetChannel(events chan<- struct{}) func(*Lister) {
	return func(lister *Lister) {
		lister.events = events
	}
}

func New(config *rest.Config, options ...func(*Lister)) (*Lister, error) {
	lister := &Lister{
		namespace:    "default",
		resyncPeriod: 0 * time.Second,
	}

	for _, option := range options {
		option(lister)
	}

	label := "test"

	listOptions := func(options *metav1.ListOptions) {
		options.LabelSelector = labels.Set(map[string]string{
			"test": label,
		}).AsSelector().String()
	}

	kubeclientset := kubernetes.NewForConfigOrDie(config)
	kubeInformerFactory := kubeinformers.NewFilteredSharedInformerFactory(
		kubeclientset,
		lister.resyncPeriod,
		lister.namespace,
		listOptions)

	return lister, nil
}
