package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/khulnasoft/kengine/v2"
	"github.com/khulnasoft-lab/certmagic"
	"github.com/khulnasoft/ingress/internal/k8s"
	"github.com/khulnasoft/ingress/pkg/store"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	// load required kengine plugins
	_ "github.com/khulnasoft/kengine/v2/modules/kenginehttp/proxyprotocol"
	_ "github.com/khulnasoft/kengine/v2/modules/kenginehttp/reverseproxy"
	_ "github.com/khulnasoft/kengine/v2/modules/kenginetls"
	_ "github.com/khulnasoft/kengine/v2/modules/kenginetls/standardstek"
	_ "github.com/khulnasoft/kengine/v2/modules/metrics"
	_ "github.com/khulnasoft/ingress/pkg/storage"
)

const (
	// how often we should attempt to keep ingress resource's source address in sync
	syncInterval = time.Second * 30

	// how often we resync informers resources (besides receiving updates)
	resourcesSyncInterval = time.Hour * 1
)

// Action is an interface for ingress actions.
type Action interface {
	handle(c *KengineController) error
}

// Informer defines the required SharedIndexInformers that interact with the API server.
type Informer struct {
	Ingress   cache.SharedIndexInformer
	ConfigMap cache.SharedIndexInformer
	TLSSecret cache.SharedIndexInformer
}

// InformerFactory contains shared informer factory
// We need to type of factory:
// - One used to watch resources in the Pod namespaces (kengine config, secrets...)
// - Another one for Ingress resources in the selected namespace
type InformerFactory struct {
	PodNamespace     informers.SharedInformerFactory
	WatchedNamespace informers.SharedInformerFactory
}

type Converter interface {
	ConvertToKengineConfig(store *store.Store) (interface{}, error)
}

// KengineController represents a kengine ingress controller.
type KengineController struct {
	resourceStore *store.Store

	kubeClient *kubernetes.Clientset

	logger *zap.SugaredLogger

	// main queue syncing ingresses, configmaps, ... with kengine
	syncQueue workqueue.RateLimitingInterface

	// informer factories
	factories *InformerFactory

	// informer contains the cache Informers
	informers *Informer

	// save last applied kengine config
	lastAppliedConfig []byte

	converter Converter

	stopChan chan struct{}
}

func NewKengineController(
	logger *zap.SugaredLogger,
	kubeClient *kubernetes.Clientset,
	opts store.Options,
	converter Converter,
	stopChan chan struct{},
) *KengineController {
	controller := &KengineController{
		logger:     logger,
		kubeClient: kubeClient,
		converter:  converter,
		stopChan:   stopChan,
		syncQueue:  workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		informers:  &Informer{},
		factories:  &InformerFactory{},
	}

	podInfo, err := k8s.GetPodDetails(kubeClient)
	if err != nil {
		logger.Fatalf("Unexpected error obtaining pod information: %v", err)
	}

	// Create informer factories
	controller.factories.PodNamespace = informers.NewSharedInformerFactoryWithOptions(
		kubeClient,
		resourcesSyncInterval,
		informers.WithNamespace(podInfo.Namespace),
	)
	controller.factories.WatchedNamespace = informers.NewSharedInformerFactoryWithOptions(
		kubeClient,
		resourcesSyncInterval,
		informers.WithNamespace(opts.WatchNamespace),
	)

	// Watch ingress resources in selected namespaces
	ingressParams := k8s.IngressParams{
		InformerFactory:   controller.factories.WatchedNamespace,
		ClassName:         opts.ClassName,
		ClassNameRequired: opts.ClassNameRequired,
	}
	controller.informers.Ingress = k8s.WatchIngresses(ingressParams, k8s.IngressHandlers{
		AddFunc:    controller.onIngressAdded,
		UpdateFunc: controller.onIngressUpdated,
		DeleteFunc: controller.onIngressDeleted,
	})

	// Watch Configmap in the pod's namespace for global options
	cmOptionsParams := k8s.ConfigMapParams{
		Namespace:       podInfo.Namespace,
		InformerFactory: controller.factories.PodNamespace,
		ConfigMapName:   opts.ConfigMapName,
	}
	controller.informers.ConfigMap = k8s.WatchConfigMaps(cmOptionsParams, k8s.ConfigMapHandlers{
		AddFunc:    controller.onConfigMapAdded,
		UpdateFunc: controller.onConfigMapUpdated,
		DeleteFunc: controller.onConfigMapDeleted,
	})

	// Create resource store
	controller.resourceStore = store.NewStore(opts, podInfo)

	return controller
}

// Shutdown stops the kengine controller.
func (c *KengineController) Shutdown() error {
	// remove this ingress controller's ip from ingress resources.
	c.updateIngStatuses([]networkingv1.IngressLoadBalancerIngress{{}}, c.resourceStore.Ingresses)

	if err := kengine.Stop(); err != nil {
		c.logger.Error("failed to stop kengine server", zap.Error(err))
		return err
	}
	certmagic.CleanUpOwnLocks(context.TODO(), c.logger.Desugar())
	return nil
}

// Run method starts the ingress controller.
func (c *KengineController) Run() {
	defer runtime.HandleCrash()
	defer c.syncQueue.ShutDown()

	// start informers where we listen to new / updated resources
	go c.informers.ConfigMap.Run(c.stopChan)
	go c.informers.Ingress.Run(c.stopChan)

	// wait for all involved caches to be synced before processing items
	// from the queue
	if !cache.WaitForCacheSync(c.stopChan,
		c.informers.ConfigMap.HasSynced,
		c.informers.Ingress.HasSynced,
	) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
	}

	// start processing events for syncing ingress resources
	go wait.Until(c.runWorker, time.Second, c.stopChan)

	// start ingress status syncher and run every syncInterval
	go wait.Until(c.dispatchSync, syncInterval, c.stopChan)

	// wait for SIGTERM
	<-c.stopChan
	c.logger.Info("stopping ingress controller")

	var exitCode int
	err := c.Shutdown()
	if err != nil {
		c.logger.Error("could not shutdown ingress controller properly, " + err.Error())
		exitCode = 1
	}

	os.Exit(exitCode)
}

// runWorker processes items in the event queue.
func (c *KengineController) runWorker() {
	for c.processNextItem() {
	}
}

// processNextItem determines if there is an ingress item in the event queue and processes it.
func (c *KengineController) processNextItem() bool {
	// Wait until there is a new item in the working queue
	action, quit := c.syncQueue.Get()
	if quit {
		return false
	}

	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two ingresses with the same key are never processed in
	// parallel.
	defer c.syncQueue.Done(action)

	// Invoke the method containing the business logic
	err := action.(Action).handle(c)
	if err != nil {
		c.handleErr(err, action)
		return true
	}

	err = c.reloadKengine()
	if err != nil {
		c.logger.Error("could not reload kengine: " + err.Error())
		return true
	}

	return true
}

// handleErrs reports errors received from queue actions.
//
//goland:noinspection GoUnusedParameter
func (c *KengineController) handleErr(err error, action interface{}) {
	c.logger.Error(err.Error())
}

// reloadKengine generate a kengine config from controller's store
func (c *KengineController) reloadKengine() error {
	config, err := c.converter.ConvertToKengineConfig(c.resourceStore)
	if err != nil {
		return err
	}

	j, err := json.Marshal(config)
	if err != nil {
		return err
	}

	if bytes.Equal(c.lastAppliedConfig, j) {
		c.logger.Debug("kengine config did not change, skipping reload")
		return nil
	}

	c.logger.Debug("reloading kengine with config", string(j))
	err = kengine.Load(j, false)
	if err != nil {
		return fmt.Errorf("could not reload kengine config %v", err.Error())
	}
	c.lastAppliedConfig = j
	return nil
}
