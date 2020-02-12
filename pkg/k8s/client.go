package k8s

import (
	"context"
	"fmt"
	"github.com/keikoproj/iam-manager/pkg/log"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	clientv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Client struct {
	cl  kubernetes.Interface
	dCl dynamic.Interface
}

//NewK8sClient gets the new k8s go client
func NewK8sClient() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		// Do i need to panic here?
		//How do i test this from local?
		//Lets get it from local config file
		config, err = clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	cl := &Client{
		client,
		dClient,
	}
	return cl, nil
}

//Iface defines required functions to be implemented by receivers
type Iface interface {
	IamrolesCount(ctx context.Context, ns string)
	GetConfigMap(ctx context.Context, ns string, name string)
}

//IamrolesCount function lists the "Iamrole" for a provided namespace
func (c *Client) IamrolesCount(ctx context.Context, ns string) (int, error) {
	log := log.Logger(ctx, "k8s", "client", "IamrolesCount")
	log.WithValues("namespace", ns)
	log.V(1).Info("list api call")
	iamCR := schema.GroupVersionResource{
		Group:    "iammanager.keikoproj.io",
		Version:  "v1alpha1",
		Resource: "iamroles",
	}

	roleList, err := c.dCl.Resource(iamCR).Namespace(ns).List(metav1.ListOptions{})
	if err != nil {
		log.Error(err, "unable to list iamroles resources")
		return 0, err
	}
	log.Info("Total number of roles", "count", len(roleList.Items))
	return len(roleList.Items), nil
}

func (c *Client) GetConfigMap(ctx context.Context, ns string, name string) *v1.ConfigMap {
	log := log.Logger(ctx, "k8s", "client", "GetConfigMap")
	log.WithValues("namespace", ns)
	log.Info("Retrieving config map")
	res, err := c.cl.CoreV1().ConfigMaps(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		log.Error(err, "unable to get config map")
		panic(err)
	}

	return res
}

func (c *Client) ClientInterface() kubernetes.Interface {
	return c.cl
}

func GetConfigMapInformer(ctx context.Context, nsName string, cmName string) cache.SharedIndexInformer {
	log := log.Logger(context.Background(), "pkg.k8s.client", "GetConfigMapInformer")
	clientset, err := NewK8sClient()
	if err != nil {
		log.Error(err, "failed to get clientset")
		return nil
	}

	listOptions := func(options *metav1.ListOptions) {
		options.FieldSelector = fmt.Sprintf("metadata.name=%s", cmName)
	}

	// default resync period 24 hours
	cmInformer := clientv1.NewFilteredConfigMapInformer(clientset.ClientInterface(), nsName, 24*time.Hour, cache.Indexers{}, listOptions)
	return cmInformer
}
