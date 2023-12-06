package command

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/client-go/kubernetes/scheme"

	garmoperatorv1alpha1 "github.com/mercedes-benz/garm-operator/api/v1alpha1"
)

func generateKubeClient() (*dynamic.DynamicClient, error) {

	config, err := clientcmd.BuildConfigFromFlags("", opt.kubeConfig)
	if err != nil {
		return nil, err
	}

	return dynamic.NewForConfig(config)
	// cliSet, err := dynamic.NewForConfig(config)
	//if err != nil {
	//	return nil, err
	//}
	//
	//// add the garm-operator CRD scheme
	//garmoperatorv1alpha1.AddToScheme(scheme.Scheme)
	//
	//return kubernetes.NewForConfig(config)
}

func newRestClient() (*rest.RESTClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", opt.kubeConfig)
	if err != nil {
		return nil, err
	}

	config.APIPath = "/apis"
	config.ContentConfig.GroupVersion = &garmoperatorv1alpha1.GroupVersion
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	return rest.RESTClientFor(config)
}
