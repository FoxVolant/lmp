package user

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/spf13/cobra"
	"lmp/eBPF_Supermarket/kernel_and_user_pod_observation/cmd/monitor/user/cilium_ebpf_probe/cluster_utils"
	"lmp/eBPF_Supermarket/kernel_and_user_pod_observation/cmd/monitor/user/cilium_ebpf_probe/http2_tracing"
	"lmp/eBPF_Supermarket/kernel_and_user_pod_observation/data"
)

func NewMonitorUserGRPCCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "grpc",
		Short:   "Starts monitor for pod by GRPC probes.",
		Long:    "",
		Example: "kupod monitor user http --pod sidecar-demo",
		RunE:    MonitorUserGRPC,
	}

	return cmd
}

func MonitorUserGRPC(cmd *cobra.Command, args []string) error {

	//1.与集群连接
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", data.Kubeconfig)
	//通过参数（master的url或者kubeconfig路径）和BuildConfigFromFlags方法来获取rest.Config对象，
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	//通过*rest.Config参数和NewForConfig方法来获取clientset对象，clientset是多个client的集合，每个client可能包含不同版本的方法调用
	if err != nil {
		panic(err.Error())
	}

	//2.获取所有的pod
	pods, err := clientset.CoreV1().Pods(data.NameSpace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d pods in the cluster in wyw namespace\n", len(pods.Items))

	//3.http2协议
	/*******uprobe on pod************/
	binaryPath := "/go/src/grpc_server/main"
	p2, err := clientset.CoreV1().Pods(data.NameSpace).Get(context.TODO(), data.GrpcPodName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Pod %s in namespace %s not found\n", data.GrpcPodName, data.NameSpace)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting pod %s in namespace %s: %v\n",
			data.GrpcPodName, data.NameSpace, statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
	} else {
		fmt.Printf("Found pod %s in namespace %s\n", data.GrpcPodName, data.NameSpace)
		res, _ := cluster_utils.GetPodELFPath(clientset, data.NodeName, data.NameSpace, data.GrpcPodName, p2.Status.ContainerStatuses, data.GrpcImageName)
		for k, v := range res {
			fmt.Printf("get pod %s Merge Path and Attach Uprobe\n", k.Name)
			go http2_tracing.GetHttp2ViaUprobe(v+binaryPath, data.GrpcPodName, data.PrometheusIP)
		}
	}

	//waiting for datas
	select {}

	return nil
}
