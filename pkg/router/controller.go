package router

//iptables -t nat -A POSTROUTING -s 10.244.1.0/24 -j MASQUERADE

import (
	"context"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func NewRouteController(log *logrus.Logger, nodeName string) error {
	//kubernetesEndpoint := "https://192.168.1.84:6443"
	namespaceName := "kube-system"
	routingTableConfigMapName := "yarp-routing-table"
	podSubnet := "10.244.0.0/24"

	config, _ := clientcmd.BuildConfigFromFlags("", "/root/.kube/config")
	clientset, _ := kubernetes.NewForConfig(config)

	routingTable, err := clientset.CoreV1().ConfigMaps(namespaceName).Get(context.Background(), routingTableConfigMapName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		configMapData := make(map[string]string, 0)
		configMapData[nodeName] = podSubnet
		configMap := corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      routingTableConfigMapName,
				Namespace: namespaceName,
			},
		}
		_, err := clientset.CoreV1().ConfigMaps(namespaceName).Create(context.Background(), &configMap, metav1.CreateOptions{})
		if err != nil {
			log.Errorf("unable to create configmap [%s]", routingTableConfigMapName)
			return err
		}
	}

	//Check if node is present
	_, ok := routingTable.Data[nodeName]
	if !ok {
		routingTable.Data[nodeName] = podSubnet
		_, err := clientset.CoreV1().ConfigMaps("game").Update(context.Background(), routingTable, metav1.UpdateOptions{})
		if err != nil {
			log.Errorf("unable to update configmap [%s] with [%s] node info", routingTableConfigMapName, nodeName)
			return err
		}
		log.Infof("Configmap [%s] updated with node [%s]", routingTableConfigMapName, nodeName)
	}

	return nil
}
