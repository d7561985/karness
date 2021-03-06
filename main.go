package main

import (
	"flag"
	"time"

	"github.com/d7561985/karness/pkg/controllers/kube"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	clientset "github.com/d7561985/karness/pkg/generated/clientset/versioned"
	informers "github.com/d7561985/karness/pkg/generated/informers/externalversions"
	"github.com/d7561985/karness/pkg/signals"
)

var (
	masterURL  string
	kubeconfig string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	client, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building example clientset: %s", err.Error())
	}

	informerFactory := informers.NewSharedInformerFactory(client, time.Second*30)

	c := kube.New(client,
		informerFactory.Karness().V1alpha1().Scenarios())

	informerFactory.Start(stopCh)

	if err = c.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controllers: %s", err.Error())
	}
}
