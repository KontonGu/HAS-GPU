/*
Copyright 2024 HAS-Function Authors, KontonGu (Jianfeng Gu), et. al.
@Techinical University of Munich, CAPS Cloud Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	fastfuncv1 "fastgshare/fastfunc/api/v1"

	fastpodv1 "github.com/KontonGu/FaST-GShare/pkg/apis/fastgshare.caps.in.tum/v1"
	fastpodclientset "github.com/KontonGu/FaST-GShare/pkg/client/clientset/versioned"
	fastpodinformer "github.com/KontonGu/FaST-GShare/pkg/client/informers/externalversions"
	fastpodlisters "github.com/KontonGu/FaST-GShare/pkg/client/listers/fastgshare.caps.in.tum/v1"
	"github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	HASGPUNodeName  = "fastgshare/nodeName"
	HASGPUNodeRole  = "fastgshare/role"
	HASvGPUID       = "fastgshare/vgpu_id"
	HASGPUTypeName  = "fastgshare/vgpu_type"
	HASBatchSize    = "has-gpu/batch_size"
	HASSchedNode    = "hasfunc/sched_node"
	HASSchedvGPU    = "hasfunc/sched_vgpu"
	HASPODNAMELABEL = "hasfunc/pod_name"
	V100GPUMem      = 16384
	RandomStrLen    = 10
)

// FaSTFuncReconciler reconciles a FaSTFunc object
type FaSTFuncReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	promv1api     promv1.API
	fastpodLister fastpodlisters.FaSTPodLister
}

type FaSTPodConfig struct {
	Quota       int64 // percentage
	SMPartition int64 // percentage
	Mem         int64
	Replicas    int64
	NodeID      string
	vGPUID      string
	BatchSize   int
	PodName     string
}

type HASPodInfo struct {
	FuncName    string
	NodeName    string
	GPUID       string
	PodName     string
	SMPartition int64
	Quota       int64
	Batch       int64
	Replica     int64
	RPS         float64
	RowIdx      int
	Memory      int64
}

type GPUInfo struct {
	vGPUID   string  // vGPU ID
	Model    string  // type
	MemoryMB float64 // GPU memory size
}

type HASResConfig struct {
	Name        string // podname
	SMPartition int64
	Quota       int64
	Memory      int64
}

type HASGPUInfo struct {
	NodeName   string
	GPUID      string
	SMConfigs  []int64                 // current sm configuration types, RowIdx -> sm value, -1 means no pod running
	HGO        float64                 // HAS GPU Occupancy, sum(s_i * q_i)
	RowPods    [][]string              // the pods in each sm configuration row, RowIdx -> Name list of pods
	PodsConfig map[string]HASResConfig // the pods's resource configuration, [pod_name] -> HASResConfig
}

type HASAction struct {
	ActionType int         // -1: delete, 0: update, 1: create
	PodItem    *HASPodInfo // mainly update GPU Quota in HASPodInfo
}

var (
	nodeGPUMap map[string][]GPUInfo            // [nodeName]-->[gpuIdx]GPUInfo, all GPUs
	podsInfo   map[string][]HASPodInfo         // [funcName]-->[podIdx]HASPodInfo
	hasGPUs    map[string][]HASGPUInfo         // [nodename]-->[gpuIdx]HASGPUInfo, only contains used GPUs
	fastPods   map[string][]*fastpodv1.FaSTPod // [funcName]-->[podIdx]*FaSTPod, stores actual FastPod objects
)

func init() {
	nodeGPUMap = make(map[string][]GPUInfo)
	podsInfo = make(map[string][]HASPodInfo)
	hasGPUs = make(map[string][]HASGPUInfo)
	fastPods = make(map[string][]*fastpodv1.FaSTPod)
}

var once_persist_reconcile sync.Once
var once_gpu_topo sync.Once

// +kubebuilder:rbac:groups=caps.in.tum.fastgshare,resources=fastfuncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=caps.in.tum.fastgshare,resources=fastfuncs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=caps.in.tum.fastgshare,resources=fastfuncs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FaSTFunc object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *FaSTFuncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// get the topology of the cluster (can remove the 'once' runing and go coroutine for periodic topology update)
	once_gpu_topo.Do(func() {
		r.getGPUDeviceTopology(ctx)
	})

	// klog.Infof("Node Topology:")
	// get the node and GPU topology
	for key, val := range nodeGPUMap {
		klog.Infof("node:%s", key)
		for _, gpuinfo := range val {
			klog.Infof("node=%s, vgpu_id=%s", key, gpuinfo.vGPUID)
		}
	}

	once_persist_reconcile.Do(func() {
		go r.persistentReconcile()
	})

	// Handle manual deletion of HAS functions
	// Get the FaSTFunc resource
	// var hasfunc fastfuncv1.FaSTFunc
	// err := r.Get(ctx, req.NamespacedName, &hasfunc)
	// if err != nil {
	// 	if errors.IsNotFound(err) {
	// 		// The HAS function has been deleted
	// 		funcName := req.Name
	// 		klog.Infof("[HAS-Cleanup] Detected manual deletion of function %s", funcName)

	// 		// Check if we have any tracking data for this function
	// 		if pods, exists := podsInfo[funcName]; exists {
	// 			klog.Infof("[HAS-Cleanup] Found %d pods to clean up for deleted function %s", len(pods), funcName)

	// 			// Get the FastPod client for deletion operations
	// 			fastpodClient, err := fastpodclientset.NewForConfig(ctrl.GetConfigOrDie())
	// 			if err != nil {
	// 				klog.Errorf("[HAS-Cleanup] Failed to create FastPod client: %v", err)
	// 				return ctrl.Result{}, err
	// 			}

	// 			// Delete all associated FastPods
	// 			for _, pod := range pods {
	// 				podName := pod.PodName
	// 				klog.Infof("[HAS-Cleanup] Deleting FastPod %s for function %s", podName, funcName)

	// 				// Delete the FastPod
	// 				err = fastpodClient.FastgshareV1().FaSTPods("fast-gshare-fn").Delete(ctx, podName, metav1.DeleteOptions{})
	// 				if err != nil && !errors.IsNotFound(err) {
	// 					klog.Errorf("[HAS-Cleanup] Failed to delete FastPod %s: %v", podName, err)
	// 					// Continue with other pods even if one fails
	// 				} else {
	// 					klog.Infof("[HAS-Cleanup] Successfully deleted FastPod %s", podName)
	// 				}

	// 				// Clean up GPU resources
	// 				if pod.NodeName != "" && pod.GPUID != "" {
	// 					// Find the GPU in hasGPUs
	// 					if gpus, nodeExists := hasGPUs[pod.NodeName]; nodeExists {
	// 						for i, gpu := range gpus {
	// 							if gpu.GPUID == pod.GPUID {
	// 								// Remove pod from GPU tracking
	// 								if gpu.PodsConfig != nil {
	// 									delete(gpu.PodsConfig, podName)
	// 								}

	// 								// Update row pods
	// 								for rowIdx, rowPods := range gpu.RowPods {
	// 									for j, p := range rowPods {
	// 										if p == podName {
	// 											// Remove pod from row
	// 											gpu.RowPods[rowIdx] = append(rowPods[:j], rowPods[j+1:]...)
	// 											break
	// 										}
	// 									}
	// 								}

	// 								// Update HGO (HAS GPU Occupancy)
	// 								r.updateHGOForGPU(&hasGPUs[pod.NodeName][i])

	// 								klog.Infof("[HAS-Cleanup] Removed pod %s from GPU %s on node %s",
	// 									podName, pod.GPUID, pod.NodeName)
	// 								break
	// 							}
	// 						}
	// 					}
	// 				}
	// 			}

	// 			// Remove function from tracking maps
	// 			delete(podsInfo, funcName)
	// 			if _, exists := fastPods[funcName]; exists {
	// 				delete(fastPods, funcName)
	// 			}

	// 			klog.Infof("[HAS-Cleanup] Completed cleanup for deleted function %s", funcName)
	// 		} else {
	// 			klog.Infof("[HAS-Cleanup] No resources to clean up for deleted function %s", funcName)
	// 		}

	// 		// The reconciliation is complete for this deleted resource
	// 		return ctrl.Result{}, nil
	// 	}
	// 	// Error reading the object - requeue the request
	// 	klog.Errorf("Failed to get FaSTFunc: %v", err)
	// 	return ctrl.Result{}, err
	// }

	// Function exists, normal reconciliation continues
	return ctrl.Result{}, nil
}

// retrieve the GPU device topology of the cluster
func (r *FaSTFuncReconciler) getGPUDeviceTopology(ctx context.Context) error {
	var podList corev1.PodList
	klog.Infof("HAS controller is trying to get GPU device topology.")
	labelSelector := client.MatchingLabels{HASGPUNodeRole: "dummyPod"}
	if err := r.Client.List(ctx, &podList, labelSelector); err != nil {
		klog.Error(err, "Failed to list Pods with specific label")
		return err
	}
	klog.Infof("konton_test--- len(podList)=%d.", len(podList.Items))
	for _, pod := range podList.Items {
		labels := pod.Labels
		node_name := labels[HASGPUNodeName]
		gpu_type := labels[HASGPUTypeName]
		vgpu_id := labels[HASvGPUID]
		nodeGPUMap[node_name] = append(nodeGPUMap[node_name], GPUInfo{vGPUID: vgpu_id, Model: gpu_type, MemoryMB: V100GPUMem})
		hasGPUs[node_name] = append(hasGPUs[node_name], HASGPUInfo{NodeName: node_name, GPUID: vgpu_id, HGO: float64(0.0)})
	}
	return nil
}

// Based on Resource Configuration and create corresponding fastpods' specification for FaSTFunc `fastfunc`
func (r *FaSTFuncReconciler) configs2FaSTPods(fastfunc *fastfuncv1.FaSTFunc, configs []*FaSTPodConfig) ([]*fastpodv1.FaSTPod, error) {
	fastpodlist := make([]*fastpodv1.FaSTPod, 0)
	for _, config := range configs {
		klog.Infof("Trying to update the FaSTPod %s with config %s with replica = %d.", fastfunc.Name, getResKeyName(config.Quota, config.SMPartition), config.Replicas)
		podSpec := corev1.PodSpec{}
		selector := metav1.LabelSelector{}

		if &fastfunc.Spec != nil {
			fastfunc.Spec.PodSpec.DeepCopyInto(&podSpec)
			fastfunc.Spec.Selector.DeepCopyInto(&selector)
		}
		extendedAnnotations := make(map[string]string)
		extendedLabels := make(map[string]string)

		// write the spec
		quota := fmt.Sprintf("%0.2f", float64(config.Quota)/100.0)
		smPartition := strconv.Itoa(int(config.SMPartition))
		mem := strconv.Itoa(int(config.Mem))
		extendedLabels["com.openfaas.scale.min"] = strconv.Itoa(int(1))
		extendedLabels["com.openfaas.scale.max"] = strconv.Itoa(int(1))
		extendedLabels["fast_function"] = fastfunc.ObjectMeta.Name
		extendedAnnotations[fastpodv1.FaSTGShareGPUQuotaRequest] = quota
		extendedAnnotations[fastpodv1.FaSTGShareGPUQuotaLimit] = quota
		extendedAnnotations[fastpodv1.FaSTGShareGPUSMPartition] = smPartition
		extendedAnnotations[fastpodv1.FaSTGShareGPUMemory] = mem
		extendedAnnotations[HASSchedNode] = config.NodeID
		extendedAnnotations[HASSchedvGPU] = config.vGPUID
		extendedAnnotations[HASBatchSize] = strconv.Itoa(int(config.BatchSize))

		// Create the fastpod name
		fastpodName := fastfunc.ObjectMeta.Name + getResKeyName(config.Quota, config.SMPartition) + "-" + generateRandomString(4)

		// Add additional matchLabels to the selector
		if selector.MatchLabels == nil {
			selector.MatchLabels = make(map[string]string)
		}
		selector.MatchLabels["app"] = fastpodName
		selector.MatchLabels["controller"] = fastpodName

		fixedReplica_int32 := int32(1)
		fastpod := &fastpodv1.FaSTPod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        fastpodName,
				Namespace:   "fast-gshare-fn",
				Labels:      extendedLabels,
				Annotations: extendedAnnotations,
			},
			Spec: fastpodv1.FaSTPodSpec{
				Selector: &selector,
				PodSpec:  podSpec,
				Replicas: &fixedReplica_int32,
			},
		}
		// ToDo: SetControllerReference here is useless, as the controller delete svc upon trial completion
		// Add owner reference to the service so that it could be GC
		if err := controllerutil.SetControllerReference(fastfunc, fastpod, r.Scheme); err != nil {
			klog.Info("Error setting ownerref")
			return nil, err
		}
		fastpodlist = append(fastpodlist, fastpod)
	}
	return fastpodlist, nil
}

// Initialize the FaSTPod Lister
func getFaSTPodLister(client fastpodclientset.Interface, namespace string, stopCh chan struct{}) fastpodlisters.FaSTPodLister {
	// create a shared informer factory for the FaasShare API group
	informerFactory := fastpodinformer.NewSharedInformerFactoryWithOptions(
		client,
		0,
		fastpodinformer.WithNamespace(namespace),
	)
	// retrieve the shared informer for FaSTPods
	fastpodInformer := informerFactory.Fastgshare().V1().FaSTPods().Informer()
	informerFactory.Start(stopCh)
	if !cache.WaitForCacheSync(stopCh, fastpodInformer.HasSynced) {
		return nil
	}
	// create a lister for FaSTPods using the shared informer's indexers
	fastpodLister := fastpodlisters.NewFaSTPodLister(fastpodInformer.GetIndexer())
	return fastpodLister
}

// SetupWithManager sets up the controller with the Manager.
func (r *FaSTFuncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create a Prometheus API client
	promClient, err := api.NewClient(api.Config{
		Address: "http://prometheus.fast-gshare.svc.cluster.local:9090",
	})
	if err != nil {
		klog.Error("Failed to create the Prometheus client.")
		return err
	}
	r.promv1api = promv1.NewAPI(promClient)
	client, _ := fastpodclientset.NewForConfig(ctrl.GetConfigOrDie())
	stopCh := make(chan struct{})
	r.fastpodLister = getFaSTPodLister(client, "fast-gshare-fn", stopCh)
	fastpodv1.AddToScheme(r.Scheme)

	return ctrl.NewControllerManagedBy(mgr).
		For(&fastfuncv1.FaSTFunc{}).
		Complete(r)
}
