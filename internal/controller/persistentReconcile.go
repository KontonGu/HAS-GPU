package controller

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	fastfuncv1 "fastgshare/fastfunc/api/v1"

	fastpodv1 "github.com/KontonGu/FaST-GShare/pkg/apis/fastgshare.caps.in.tum/v1"
	fastpodclientset "github.com/KontonGu/FaST-GShare/pkg/client/clientset/versioned"
	"github.com/prometheus/common/model"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

// func (r *FaSTFuncReconciler) persistentReconcile() {
// 	// update every 2 seconds
// 	ticker_itv := 2
// 	ticker := time.NewTicker(2 * time.Second)
// 	defer ticker.Stop()

// 	shortReqPred := ImprovedKalmanFilter(0.0, 1.0, 10.0)
// 	// longReqPred := ImprovedKalmanFilter(0.0, 1.0, 10.0)
// 	// longtermCnt := 0
// 	// longtermRange := 15 // the update interval = longtermRange * tickerTime
// 	tried := false
// 	recordWin := 12
// 	reqPreT := 0.0
// 	sampleItv := 3
// 	reqRecordList := make(map[string][]float64)
// 	reqPredList := make(map[string][]float64)
// 	for range ticker.C {
// 		ctx := context.TODO()
// 		// logger := log.FromContext(ctx)

// 		var allHASfuncs fastfuncv1.FaSTFuncList
// 		if err := r.List(ctx, &allHASfuncs); err != nil {
// 			klog.Error(err, "Failed to get FaSTFuncs.")
// 			return
// 		}
// 		// reconcile for each HAS function
// 		for _, hasfunc := range allHASfuncs.Items {

// 			//KONTON_WS
// 			funcName := hasfunc.ObjectMeta.Name
// 			// alwasy keep one instance in the cluster, scale-to-1 policy
// 			if !tried {
// 				klog.Infof("Initializing the first instance of the HASFunc %s.", funcName)
// 				tried = true
// 				config, _ := getMostEfficientConfig()
// 				config.Replicas = 3
// 				config.NodeID = "kgpu1"
// 				config.vGPUID = nodeGPUMap[config.NodeID][0].vGPUID
// 				klog.Infof("konton_test### config: %s-%s-%d.", config.NodeID, config.vGPUID, config.BatchSize)
// 				var configslist []*FaSTPodConfig
// 				configslist = append(configslist, &config)
// 				fastpods, _ := r.configs2FaSTPods(&hasfunc, configslist)
// 				err := r.Create(context.TODO(), fastpods[0])
// 				if err != nil {
// 					klog.Errorf("Failed to create the fastfunc %s.", funcName)
// 				}
// 			}

// 			// retrive the past T(T0) second's rps
// 			klog.Infof("Checking HASFunc %s.", funcName)
// 			pstRPST0 := float64(0.0)
// 			queryT0 := fmt.Sprintf("rate(gateway_function_invocation_total{function_name='%s.%s'}[15s])", funcName, hasfunc.ObjectMeta.Namespace)
// 			// klog.Infof("Prometheus Query: %s.", queryT0)
// 			queryResT0, _, err := r.promv1api.Query(ctx, queryT0, time.Now())
// 			if err != nil {
// 				klog.Errorf("Error Failed to get RPS of function %s: %s.", funcName, err.Error())
// 				continue
// 			}
// 			if queryResT0.(model.Vector).Len() != 0 {
// 				// klog.Infof("The past rps vec for function %s is %v.", funcName, queryResT0)
// 				pstRPST0 = float64(queryResT0.(model.Vector)[0].Value)
// 			}
// 			klog.Infof("The past [%ds], rps for function %s is %f.", recordWin, funcName, pstRPST0)

// 			// retrive the past T0-2(T1) second's rps
// 			pstRPST1 := float64(0.0)
// 			queryT1 := fmt.Sprintf("rate(gateway_function_invocation_total{function_name='%s.%s'}[13s])", funcName, hasfunc.ObjectMeta.Namespace)
// 			// klog.Infof("Prometheus Query: %s.", queryT1)
// 			queryResT1, _, err := r.promv1api.Query(ctx, queryT1, time.Now())
// 			if err != nil {
// 				klog.Errorf("Error Failed to get RPS of function %s: %s.", funcName, err.Error())
// 				continue
// 			}
// 			if queryResT1.(model.Vector).Len() != 0 {
// 				// klog.Infof("The past rps vec for function %s is %v.", funcName, queryResT1)
// 				pstRPST1 = float64(queryResT1.(model.Vector)[0].Value)
// 			}
// 			klog.Infof("The past [%ds], rps for function %s is %f.", recordWin-sampleItv, funcName, pstRPST1)

// 			actualRPS := float64(0.0)
// 			actualRPS = (pstRPST0*float64(recordWin) - reqPreT*(float64(recordWin-sampleItv))) / (float64(sampleItv))
// 			actualRPS = math.Max(actualRPS, 0)
// 			reqPreT = pstRPST1
// 			reqRecordList[funcName] = append(reqRecordList[funcName], actualRPS)
// 			// the request prediction in the next 2s
// 			predRPS := shortReqPred.Update(actualRPS)
// 			reqPredList[funcName] = append(reqPredList[funcName], predRPS)

// 			klog.Infof("The future[%ds], the predicted rps for function %s is %f.", ticker_itv, funcName, actualRPS)
// 			klog.Infof("-------------------------------------------------------------------")
// 			// klog.Infof("Reqs list: %v.", reqRecordList)
// 			// klog.Infof("Pred list: %v.", reqPredList)
// 			strReqs := make([]string, len(reqRecordList[funcName]))
// 			for i, v := range reqRecordList[funcName] {
// 				// 将 v 转为带两位小数的字符串
// 				strReqs[i] = fmt.Sprintf("%.2f", v)
// 			}
// 			klog.Infof("Reqs list: %v", strReqs)

// 			strPreds := make([]string, len(reqPredList[funcName]))
// 			for i, v := range reqPredList[funcName] {
// 				strPreds[i] = fmt.Sprintf("%.2f", v)
// 			}
// 			klog.Infof("Pred list: %v", strPreds)
// 		}
// 	}
// }

func (r *FaSTFuncReconciler) persistentReconcile() {
	// update every 5 seconds
	ticker_itv := 2
	ticker := time.NewTicker(time.Duration(ticker_itv) * time.Second)
	defer ticker.Stop()

	shortReqPred := ImprovedKalmanFilter(0.0, 1.0, 5.0)
	//longReqPred := ImprovedKalmanFilter(0.0, 1.0, 10.0)
	// longtermCnt := 0
	// longtermRange := 15 // the update interval = longtermRange * tickerTime
	recordWin := 10
	reqRecordList := make(map[string][]float64)
	reqPredList := make(map[string][]float64)

	// Directory to store GPU usage data for cost calculation
	costDataDir := "/data/gpu_usage"
	// Create directory if it doesn't exist
	if err := os.MkdirAll(costDataDir, 0755); err != nil {
		klog.Errorf("Failed to create directory for GPU usage data: %v", err)
	}

	// Initialize cost data file
	costDataFile := fmt.Sprintf("%s/has_gpu_usage_data.csv", costDataDir)
	// Create or truncate the file with headers
	headers := "timestamp,node_name,gpu_id,pod_name,function_name,sm_partition,quota,memory,hgo\n"
	if err := os.WriteFile(costDataFile, []byte(headers), 0644); err != nil {
		klog.Errorf("Failed to initialize GPU usage data file: %v", err)
	}

	for range ticker.C {
		ctx := context.TODO()
		// logger := log.FromContext(ctx)

		var allHASfuncs fastfuncv1.FaSTFuncList
		if err := r.List(ctx, &allHASfuncs); err != nil {
			klog.Error(err, "Failed to get FaSTFuncs.")
			return
		}
		// reconcile for each HAS function
		for _, hasfunc := range allHASfuncs.Items {

			//KONTON_WS
			funcName := hasfunc.ObjectMeta.Name
			// alwasy keep one instance in the cluster, scale-to-1 policy, initialization （just specify
			// the required throughput, and use auto-scaling to schedule GPU for instances
			if _, ok := podsInfo[funcName]; !ok {
				// Create a placeholder entry to prevent multiple initialization attempts
				podsInfo[funcName] = []HASPodInfo{}

				// Check if there are already FastPods for this function before creating one
				podCount, existingPods, readyPodCount, err := r.checkFaSTPodsForFunction(funcName)
				if err != nil {
					klog.Errorf("Error checking if FastPods exist for function %s: %v", funcName, err)
					continue
				}

				// Only initialize if no pods exist
				if podCount > 0 {
					klog.Infof("Found %d existing FastPods (%d ready) for function %s, skipping initialization",
						podCount, readyPodCount, funcName)

					// Update our tracking maps with the existing pods
					fastPods[funcName] = existingPods

					// Convert to HASPodInfo for our tracking
					podsInfoPtr, err := r.convertFastPodsToHASPodInfos(existingPods)
					if err != nil {
						klog.Errorf("Error converting existing FastPods to HASPodInfo for function %s: %v", funcName, err)
						continue
					}

					// Update podsInfo map
					podsInfo[funcName] = make([]HASPodInfo, len(podsInfoPtr))
					for i, podInfo := range podsInfoPtr {
						podsInfo[funcName][i] = *podInfo

						// Also update hasGPUs map
						r.addPodtoHASGPU(&podsInfo[funcName][i], hasGPUs)
					}

					klog.Infof("Updated tracking information for %d existing pods for function %s", len(podsInfoPtr), funcName)
					continue
				} else {
					klog.Infof("Initializing the first instance of the HASFunc %s.", funcName)

					// Special handling for rapp functions
					var smPartition, quota int64
					var batchSize int
					
					if strings.Contains(strings.ToLower(funcName), "rapp") {
						// Use predefined values for rapp functions
						smPartition = RappDefaultSM
						quota = RappDefaultQuota
						batchSize = int(RappDefaultBatchSize)
						
						klog.Infof("Using predefined configuration for rapp function: SM=%d, Quota=%d, BatchSize=%d",
							smPartition, quota, batchSize)
					} else {
						// Get the most efficient configuration for other functions
						smPartition, quota, batchSize = r.getEfficientConfigSafely(funcName)
					}

					// Create a new configuration
					config := FaSTPodConfig{
						SMPartition: smPartition,
						Quota:       quota,
						BatchSize:   batchSize,
						Mem:         3700000000,
					}

					// Set replicas to 1 initially - let autoscaling handle the rest
					config.Replicas = 1

					// Find a GPU with low HGO (HAS GPU Occupancy)
					lowHGOGPUs := RetrieveUsedGPUOrderedByHGO(hasGPUs, 0.5) // GPUs with < 50% occupancy

					var selectedNode string
					var selectedGPU string

					if len(lowHGOGPUs) > 0 {
						// Use the GPU with the lowest HGO
						selectedNode = lowHGOGPUs[0].NodeName
						selectedGPU = lowHGOGPUs[0].GPUID
					} else {
						// If no low HGO GPUs, try to find completely unused GPUs
						availableGPUs := RetrieveNewGPU(hasGPUs)
						if len(availableGPUs) > 0 {
							// Use the first available node and GPU
							for node, gpus := range availableGPUs {
								if len(gpus) > 0 {
									selectedNode = node
									selectedGPU = gpus[0]
									break
								}
							}
						} else {
							// As a last resort, use the first available node/GPU from nodeGPUMap
							for node, gpuInfos := range nodeGPUMap {
								if len(gpuInfos) > 0 {
									selectedNode = node
									selectedGPU = gpuInfos[0].vGPUID
									break
								}
							}
						}
					}

					// Set the selected node and GPU
					config.NodeID = selectedNode
					config.vGPUID = selectedGPU

					// Find the GPU in the hasGPUs map
					var gpuInfo HASGPUInfo
					var gpuExists bool = false

					// Check if the node exists in hasGPUs
					if _, exists := hasGPUs[config.NodeID]; exists {
						// Find the GPU in the node
						for _, gpu := range hasGPUs[config.NodeID] {
							if gpu.GPUID == config.vGPUID {
								gpuInfo = gpu
								gpuExists = true
								break
							}
						}
					}

					// Determine the optimal row index for the new pod
					var rowIdx int = 0
					if gpuExists {
						// Use the FindOptimalRowIdx function to get the best row
						rowIdx = FindOptimalRowIdx(gpuInfo, config.SMPartition)
					}

					// Create HASPodInfo before creating the actual pod
					tmphasPod := HASPodInfo{
						FuncName: funcName,
						NodeName: config.NodeID,
						GPUID:    config.vGPUID,
						//PodName:     funcName + getResKeyName(config.Quota, config.SMPartition),
						SMPartition: config.SMPartition,
						Quota:       config.Quota,
						Batch:       int64(config.BatchSize),
						Replica:     config.Replicas,
						RowIdx:      rowIdx,
					}

					klog.Infof("[HAS-Autoscaling] Initial config for %s: Node=%s, GPU=%s, BatchSize=%d, SMPartition=%d, Quota=%d, RowIdx=%d",
						funcName, config.NodeID, config.vGPUID, config.BatchSize, config.SMPartition, config.Quota, rowIdx)

					// Create the FastPod configuration list
					var configslist []*FaSTPodConfig
					configslist = append(configslist, &config)

					// Generate the FastPod objects
					fastpods, err := r.configs2FaSTPods(&hasfunc, configslist)

					tmphasPod.PodName = fastpods[0].Name

					// Only try to create the pod if we got valid fastpods back
					//if err == nil && len(fastpods) > 0 {
					// Check if the pod already exists before trying to create it
					// existingPod := &fastpodv1.FaSTPod{}
					// err := r.Get(context.TODO(), types.NamespacedName{
					// 	Namespace: fastpods[0].Namespace,
					// 	Name:      fastpods[0].Name,
					// }, existingPod)

					//if err != nil && errors.IsNotFound(err) {
					// Pod doesn't exist, create it
					err = r.Create(context.TODO(), fastpods[0])
					if err != nil {
						klog.Errorf("Failed to create the fastfunc %s: %v.", funcName, err)
					} else {
						klog.Infof("Successfully created new pod for function %s.", funcName)

						// Now that the pod is created, update our tracking structures

						// Add the pod to hasGPUs
						r.addPodtoHASGPU(&tmphasPod, hasGPUs)

						// Store the pod info in podsInfo map
						if _, exists := podsInfo[funcName]; !exists {
							podsInfo[funcName] = []HASPodInfo{}
						}
						podsInfo[funcName] = append(podsInfo[funcName], tmphasPod)

						klog.Infof("[HAS-Autoscaling] Initialized pod %s for function %s on GPU %s with SM partition %d and quota %d",
							fastpods[0].Name, funcName, config.vGPUID, config.SMPartition, config.Quota)
					}
					// } else if err != nil {
					// 	klog.Errorf("Error checking if pod exists for function %s: %v", funcName, err)
					// } else {
					// 	klog.Infof("Pod %s already exists for function %s, skipping creation.", fastpods[0].Name, funcName)
					// }
					//} else {
					//	if err != nil {
					//		klog.Errorf("Failed to generate FaSTPods for function %s: %v", funcName, err)
					//	} else {
					//		klog.Errorf("No FaSTPods were generated for function %s", funcName)
					//	}
					//}

				}
			}

			// Skip all processing for rapp functions
			if strings.Contains(strings.ToLower(funcName), "rapp") {
				klog.Info("Skipping prediction and autoscaling for rapp function")
				continue
			}

			// retrive the past T(T0) second's rps
			klog.Infof("Checking HASFunc %s.", funcName)
			pstRPST0 := float64(0.0)
			queryT0 := fmt.Sprintf("rate(gateway_function_invocation_total{function_name='%s.%s'}[10s])", funcName, hasfunc.ObjectMeta.Namespace)
			// klog.Infof("Prometheus Query: %s.", queryT0)
			queryResT0, _, err := r.promv1api.Query(ctx, queryT0, time.Now())
			if err != nil {
				klog.Errorf("Error Failed to get RPS of function %s: %s.", funcName, err.Error())
				continue
			}
			if queryResT0.(model.Vector).Len() != 0 {
				// klog.Infof("The past rps vec for function %s is %v.", funcName, queryResT0)
				pstRPST0 = float64(queryResT0.(model.Vector)[0].Value)
			}
			klog.Infof("The past [%ds], rps for function %s is %f.", recordWin, funcName, pstRPST0)

			actualRPS := pstRPST0
			reqRecordList[funcName] = append(reqRecordList[funcName], actualRPS)
			// the request prediction in the next 2s
			predRPS := shortReqPred.Update(actualRPS)
			reqPredList[funcName] = append(reqPredList[funcName], predRPS)

			klog.Infof("The future[%ds], the predicted rps for function %s is %f.", ticker_itv, funcName, actualRPS)
			klog.Infof("-------------------------------------------------------------------")
			// klog.Infof("Reqs list: %v.", reqRecordList)
			// klog.Infof("Pred list: %v.", reqPredList)
			strReqs := make([]string, len(reqRecordList[funcName]))
			for i, v := range reqRecordList[funcName] {
				strReqs[i] = fmt.Sprintf("%.2f", v)
			}
			klog.Infof("Reqs list: %v", strReqs)

			strPreds := make([]string, len(reqPredList[funcName]))
			for i, v := range reqPredList[funcName] {
				strPreds[i] = fmt.Sprintf("%.2f", v)
			}
			klog.Infof("Pred list: %v", strPreds)

			// Connect to autoscaling functions
			// Check if we have any pods for this function
			if podsList, exists := podsInfo[funcName]; exists && len(podsList) > 0 {
				// Check if there are any actual FastPods running for this function
				podCount, functionFastPods, readyPodCount, err := r.checkFaSTPodsForFunction(funcName)
				if err != nil {
					klog.Errorf("[HAS-Autoscaling] Error checking FastPods for function %s: %v", funcName, err)
					continue
				}

				// Skip autoscaling if no FastPods are found, even if podsInfo has entries
				if podCount == 0 || (podCount > 0 && readyPodCount == 0) {
					klog.Infof("[HAS-Autoscaling] No actual FastPods found for function %s despite having podsInfo entries. Skipping autoscaling.", funcName)
					continue
				}

				// Update the fastPods map with the actual FastPod objects
				fastPods[funcName] = functionFastPods

				klog.Infof("[HAS-Autoscaling] Starting autoscaling process for function %s with predicted RPS %.2f", funcName, predRPS)

				// Convert FastPod objects to HASPodInfo objects for compatibility with existing autoscaling logic
				podsInfoPtr, err := r.convertFastPodsToHASPodInfos(functionFastPods)
				if err != nil {
					klog.Errorf("[HAS-Autoscaling] Error converting FastPods to HASPodInfo for function %s: %v", funcName, err)
					continue
				}

				// Log current pod and GPU state before autoscaling
				klog.Infof("[HAS-Autoscaling] Current state for function %s: %d pods, total capacity: %.2f RPS",
					funcName, len(podsInfoPtr), r.GetFuncRPSCapability(podsInfoPtr))

				// Call the autoscaling function with the predicted RPS
				actions := r.HASFuncAutoScale(predRPS, podsInfoPtr, hasGPUs)

				// Process the actions returned by the autoscaling function
				if len(actions) > 0 {
					klog.Infof("[HAS-Autoscaling] Autoscaling generated %d actions for function %s", len(actions), funcName)

					// Initialize counters for action types
					createCount := 0
					updateCount := 0
					deleteCount := 0

					// Process each action
					for _, action := range actions {
						switch action.ActionType {
						case -1: // Delete pod
							deleteCount++
							// Find and delete the pod
							podName := action.PodItem.PodName

							// Find the FastPod directly from our map
							var fastPodToDelete *fastpodv1.FaSTPod
							for _, fp := range fastPods[funcName] {
								if fp.Name == podName {
									fastPodToDelete = fp
									break
								}
							}

							if fastPodToDelete == nil {
								klog.Errorf("[HAS-Autoscaling] Error finding FastPod %s for deletion", podName)
								continue
							}

							// Get the FastPod client
							fastpodClient, err := fastpodclientset.NewForConfig(ctrl.GetConfigOrDie())
							if err != nil {
								klog.Errorf("[HAS-Autoscaling] Failed to create FastPod client: %v", err)
								continue
							}

							klog.Infof("[HAS-Autoscaling] Deleting FastPod %s for function %s", podName, funcName)
							err = fastpodClient.FastgshareV1().FaSTPods(fastPodToDelete.Namespace).Delete(context.Background(), podName, metav1.DeleteOptions{})
							if err != nil {
								klog.Errorf("[HAS-Autoscaling] Failed to delete FastPod %s: %v", podName, err)
							} else {
								// Remove FastPod from our map
								updatedFastPods := make([]*fastpodv1.FaSTPod, 0, len(fastPods[funcName])-1)
								for _, fp := range fastPods[funcName] {
									if fp.Name != podName {
										updatedFastPods = append(updatedFastPods, fp)
									}
								}
								fastPods[funcName] = updatedFastPods

								// Also remove from podsInfo for backward compatibility
								found := false
								for i, p := range podsInfo[funcName] {
									if p.PodName == podName {
										podsInfo[funcName] = append(podsInfo[funcName][:i], podsInfo[funcName][i+1:]...)
										found = true
										break
									}
								}
								if !found {
									klog.Warningf("[HAS-Autoscaling] Pod %s was not found in podsInfo after deletion", podName)
								}
							}

						case 0: // Update pod (vertical scaling)
							updateCount++
							// Find the FastPod directly from our map
							podName := action.PodItem.PodName
							newQuota := action.PodItem.Quota
							// Ensure quota is at least 5
							if newQuota < 5 {
								newQuota = 5
							}

							// Find the FastPod directly from our map
							var fastPodToUpdate *fastpodv1.FaSTPod
							for _, fp := range fastPods[funcName] {
								if fp.Name == podName {
									fastPodToUpdate = fp
									break
								}
							}

							if fastPodToUpdate == nil {
								klog.Errorf("[HAS-Autoscaling] Error finding FastPod %s for update", podName)
								continue
							}

							// Create a deep copy to avoid modifying the cache
							fastPodCopy := fastPodToUpdate.DeepCopy()

							// Get the SM partition from the existing FastPod
							for _, p := range podsInfo[funcName] {
								if p.PodName == podName {
									smPartition := p.SMPartition
									batchSize := p.Batch

									// Calculate the new RPS capacity based on the updated quota
									newRPSCapacity := rapp(funcName, batchSize, smPartition, newQuota)

									// Generate the new FastPod name with the updated quota
									//newFastPodName := funcName + getResKeyName(newQuota, smPartition)

									klog.Infof("[HAS-Autoscaling] Vertical scaling: Creating new FastPod %s to replace %s with quota %d (new capacity: %.2f RPS)",
										funcName, podName, newQuota, newRPSCapacity)

									// Update the quota annotation
									quota := fmt.Sprintf("%0.2f", float64(newQuota)/100.0)
									if fastPodCopy.Annotations == nil {
										fastPodCopy.Annotations = make(map[string]string)
									}
									fastPodCopy.Annotations[fastpodv1.FaSTGShareGPUQuotaRequest] = quota
									fastPodCopy.Annotations[fastpodv1.FaSTGShareGPUQuotaLimit] = quota

									// Update the name of the FastPod
									//fastPodCopy.ObjectMeta.Name = newFastPodName

									// Get the FastPod client
									fastpodClient, err := fastpodclientset.NewForConfig(ctrl.GetConfigOrDie())
									if err != nil {
										klog.Errorf("[HAS-Autoscaling] Failed to create FastPod client: %v", err)
										continue
									}

									//revision
									//fastPodCopy.ObjectMeta.ResourceVersion = ""

									// Update the FastPod
									createdFastPod, err := fastpodClient.FastgshareV1().FaSTPods(fastPodCopy.Namespace).Update(context.Background(), fastPodCopy, metav1.UpdateOptions{})
									if err != nil {
										klog.Errorf("[HAS-Autoscaling] Failed to update FastPod %s: %v", fastPodCopy.ObjectMeta.Name, err)
									} else {
										klog.Infof("[HAS-Autoscaling] Successfully updated FastPod %s with quota %s", fastPodCopy.ObjectMeta.Name, quota)

										// Update the FastPod in our map
										var updatedFastPods []*fastpodv1.FaSTPod
										for _, fp := range fastPods[funcName] {
											if fp.Name != podName {
												updatedFastPods = append(updatedFastPods, fp)
											}
										}
										updatedFastPods = append(updatedFastPods, createdFastPod)
										fastPods[funcName] = updatedFastPods

										// Also update in podsInfo for backward compatibility
										found := false
										for i, p := range podsInfo[funcName] {
											if p.PodName == podName {
												// Update the quota and RPS capacity
												podsInfo[funcName][i].Quota = newQuota
												podsInfo[funcName][i].RPS = newRPSCapacity
												found = true
												break
											}
										}
										if !found {
											klog.Warningf("[HAS-Autoscaling] Pod %s was not found in podsInfo after update", podName)
										}

										// Update the hasGPUs structure to reflect the changes
										for nodeName, gpus := range hasGPUs {
											for i, gpu := range gpus {
												if gpu.GPUID == action.PodItem.GPUID && nodeName == action.PodItem.NodeName {
													if podConfig, exists := hasGPUs[nodeName][i].PodsConfig[podName]; exists {
														// Update the quota in the PodsConfig
														oldQuota := podConfig.Quota
														podConfig.Quota = newQuota
														hasGPUs[nodeName][i].PodsConfig[podName] = podConfig

														// Recalculate HGO based on the quota change
														oldHGO := hasGPUs[nodeName][i].HGO
														// Formula: HGO += (SMPartition/100.0) * (newQuota - oldQuota) / 100.0
														hasGPUs[nodeName][i].HGO = oldHGO + float64((podConfig.SMPartition/100.0)*(newQuota-oldQuota)/100.0)

														klog.Infof("[HAS-Autoscaling] Updated GPU %s on node %s HGO: %.4f -> %.4f due to quota change: %d -> %d",
															gpu.GPUID, nodeName, oldHGO, hasGPUs[nodeName][i].HGO, oldQuota, newQuota)
													}
													break
												}
											}
										}
									}
								}
							}

						case 1: // Create fast pod (horizontal scaling)
							createCount++
							// Create a new pod based on the configuration
							podInfo := action.PodItem

							// Validate pod information
							if podInfo.NodeName == "" || podInfo.GPUID == "" {
								klog.Errorf("[HAS-Autoscaling] Invalid pod information for creation: NodeName=%s, GPUID=%s",
									podInfo.NodeName, podInfo.GPUID)
								continue
							}

							// Create a FaSTPodConfig for the new pod
							config := &FaSTPodConfig{
								SMPartition: podInfo.SMPartition,
								Quota:       podInfo.Quota,
								BatchSize:   int(podInfo.Batch),
								Mem:         3700000000, // Default memory
								NodeID:      podInfo.NodeName,
								vGPUID:      podInfo.GPUID,
								Replicas:    1,
							}

							// Get the FastFunc object for this function
							var hasfunc fastfuncv1.FaSTFunc
							err := r.Get(context.TODO(), types.NamespacedName{
								Namespace: "fast-gshare-fn",
								Name:      funcName,
							}, &hasfunc)
							if err != nil {
								klog.Errorf("[HAS-Autoscaling] Failed to get FastFunc %s: %v", funcName, err)
								continue
							}

							// Use configs2FaSTPods to create the FastPod
							configsList := []*FaSTPodConfig{config}
							fastpods, err := r.configs2FaSTPods(&hasfunc, configsList)
							if err != nil {
								klog.Errorf("[HAS-Autoscaling] Failed to generate FastPod for function %s: %v", funcName, err)
								continue
							}

							if len(fastpods) == 0 {
								klog.Errorf("[HAS-Autoscaling] No FastPods were generated for function %s", funcName)
								continue
							}

							// Get the FastPod client
							fastpodClient, err := fastpodclientset.NewForConfig(ctrl.GetConfigOrDie())
							if err != nil {
								klog.Errorf("[HAS-Autoscaling] Failed to create FastPod client: %v", err)
								continue
							}

							// Create the FastPod
							klog.Infof("[HAS-Autoscaling] Creating new FastPod %s for function %s with quota %d, SM partition %d",
								fastpods[0].Name, funcName, podInfo.Quota, podInfo.SMPartition)

							createdFastPod, err := fastpodClient.FastgshareV1().FaSTPods(fastpods[0].Namespace).Create(context.Background(), fastpods[0], metav1.CreateOptions{})
							if err != nil {
								klog.Errorf("[HAS-Autoscaling] Failed to create FastPod %s: %v", fastpods[0].Name, err)
							} else {
								klog.Infof("[HAS-Autoscaling] Successfully created FastPod %s", createdFastPod.Name)

								// Add the new FastPod to our map
								fastPods[funcName] = append(fastPods[funcName], createdFastPod)

								// Update the podInfo map with the new pod
								podInfo.PodName = createdFastPod.Name
								if _, exists := podsInfo[funcName]; !exists {
									podsInfo[funcName] = []HASPodInfo{}
								}
								podsInfo[funcName] = append(podsInfo[funcName], *podInfo)
							}
						}
					}

					// Log summary of actions
					klog.Infof("[HAS-Autoscaling] Summary for function %s: Created %d pods, Updated %d pods, Deleted %d pods",
						funcName, createCount, updateCount, deleteCount)
				} else {
					klog.Infof("[HAS-Autoscaling] No scaling actions needed for function %s", funcName)
				}
			} else {
				// No pods found for this function, just log and continue
				if !exists {
					klog.Warningf("[HAS-Autoscaling] No pods found for function %s in podsInfo map", funcName)
				} else if len(podsList) == 0 {
					klog.Warningf("[HAS-Autoscaling] Empty pods list for function %s", funcName)
				}
			}
		}

		// Record GPU usage data at the end of each reconcile cycle
		r.recordGPUUsageData(costDataFile)
	}
}

// recordGPUUsageData records current GPU usage information to a file for cost calculation
func (r *FaSTFuncReconciler) recordGPUUsageData(dataFile string) {
	// Get current timestamp
	timestamp := time.Now().Format(time.RFC3339)

	// Open file for appending
	file, err := os.OpenFile(dataFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		klog.Errorf("Failed to open GPU usage data file: %v", err)
		return
	}
	defer file.Close()

	// Write data for each GPU with detailed pod metrics
	for nodeName, gpus := range hasGPUs {
		for _, gpu := range gpus {
			// If there are no pods on this GPU, record a single line with zeros
			if len(gpu.PodsConfig) == 0 {
				line := fmt.Sprintf("%s,%s,%s,none,none,0,0,0,%.4f\n",
					timestamp, nodeName, gpu.GPUID, gpu.HGO)

				if _, err := file.WriteString(line); err != nil {
					klog.Errorf("Failed to write GPU usage data: %v", err)
					return
				}
				continue
			}

			// Record detailed information for each pod on this GPU
			for podName, podConfig := range gpu.PodsConfig {
				// Extract function name from pod name (remove quota and partition info)
				funcName := extractFunctionNameFromPod(podName)

				// Validate SM partition - ensure it's positive
				if podConfig.SMPartition <= 0 {
					// Extract partition from pod name as fallback
					partitionFromName := extractPartitionFromPodName(podName)
					if partitionFromName > 0 {
						klog.Warningf("Pod %s has invalid partition %d in PodsConfig, using %d from pod name",
							podName, podConfig.SMPartition, partitionFromName)
						podConfig.SMPartition = partitionFromName
					} else {
						klog.Warningf("Pod %s has invalid partition %d, using default value 50",
							podName, podConfig.SMPartition)
						podConfig.SMPartition = 50 // Use default value if invalid
					}
				}

				// Calculate this pod's contribution to HGO
				podHGO := float64(podConfig.SMPartition) / 100.0 * float64(podConfig.Quota) / 100.0

				// Write a line for each pod with detailed data
				line := fmt.Sprintf("%s,%s,%s,%s,%s,%d,%d,%d,%.4f\n",
					timestamp, nodeName, gpu.GPUID,
					podName, funcName,
					podConfig.SMPartition, podConfig.Quota, podConfig.Memory,
					podHGO)

				if _, err := file.WriteString(line); err != nil {
					klog.Errorf("Failed to write GPU usage data: %v", err)
					return
				}
			}
		}
	}
}

// extractFunctionNameFromPod extracts the function name from a pod name
// Pod names typically follow the pattern: funcname-qXX-pYY-ZZZZ where
// XX is quota, YY is partition, and ZZZZ is a random suffix
func extractFunctionNameFromPod(podName string) string {
	// Split by hyphens
	parts := strings.Split(podName, "-")

	// If the pod name doesn't follow the expected format, return the whole name
	if len(parts) < 3 {
		return podName
	}

	// Check if we have the q and p parts
	hasQ := false
	hasP := false
	qIndex := -1

	for i, part := range parts {
		if len(part) > 1 && part[0] == 'q' && isNumeric(part[1:]) {
			hasQ = true
			qIndex = i
			break
		}
	}

	if qIndex > 0 && qIndex+1 < len(parts) {
		part := parts[qIndex+1]
		if len(part) > 1 && part[0] == 'p' && isNumeric(part[1:]) {
			hasP = true
		}
	}

	// If we found the expected pattern, return everything before the q part
	if hasQ && hasP {
		return strings.Join(parts[:qIndex], "-")
	}

	return podName
}

// extractPartitionFromPodName extracts the partition value from a pod name
// Pod names typically follow the pattern: funcname-qXX-pYY-ZZZZ where
// XX is quota, YY is partition, and ZZZZ is a random suffix
func extractPartitionFromPodName(podName string) int64 {
	// Split by hyphens
	parts := strings.Split(podName, "-")

	// Look for the part starting with "p" (partition)
	for _, part := range parts {
		if strings.HasPrefix(part, "p") {
			// Extract the number after "p"
			partitionStr := part[1:]
			partition, err := strconv.ParseInt(partitionStr, 10, 64)
			if err == nil {
				return partition
			}
			// If parsing fails, return 0
			return 0
		}
	}

	// If no partition found, return 0
	return 0
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// checkFaSTPodsForFunction checks if there are any FastPods running for a given function
// Returns the count of FastPods, a list of FastPods, the count of ready Kubernetes pods, and an error
func (r *FaSTFuncReconciler) checkFaSTPodsForFunction(funcName string) (int, []*fastpodv1.FaSTPod, int, error) {
	// Create a FastPod client
	fastpodClient, err := fastpodclientset.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		klog.Errorf("[HAS-Autoscaling] Failed to create FastPod client: %v", err)
		return 0, nil, 0, err
	}

	// List all FastPods in the namespace
	fastPodList, err := fastpodClient.FastgshareV1().FaSTPods("fast-gshare-fn").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		klog.Errorf("[HAS-Autoscaling] Failed to list FastPods: %v", err)
		return 0, nil, 0, err
	}

	// Filter FastPods for the given function
	var functionFastPods []*fastpodv1.FaSTPod
	for i := range fastPodList.Items {
		pod := &fastPodList.Items[i]
		// Check if this pod belongs to the function
		if pod.Labels["fast_function"] == funcName {
			functionFastPods = append(functionFastPods, pod)
		}
	}

	// Create a Kubernetes client to check for actual ready pods
	k8sClient, err := ctrl.GetConfig()
	if err != nil {
		klog.Errorf("[HAS-Autoscaling] Failed to get Kubernetes config: %v", err)
		return len(functionFastPods), functionFastPods, 0, err
	}

	clientset, err := kubernetes.NewForConfig(k8sClient)
	if err != nil {
		klog.Errorf("[HAS-Autoscaling] Failed to create Kubernetes client: %v", err)
		return len(functionFastPods), functionFastPods, 0, err
	}

	// List all pods in the namespace
	podList, err := clientset.CoreV1().Pods("fast-gshare-fn").List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("fast_function=%s", funcName),
	})
	if err != nil {
		klog.Errorf("[HAS-Autoscaling] Failed to list Kubernetes pods: %v", err)
		return len(functionFastPods), functionFastPods, 0, err
	}

	// Count ready pods
	readyPodCount := 0
	for _, pod := range podList.Items {
		// Check if pod is ready
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				readyPodCount++
				break
			}
		}
	}

	klog.Infof("[HAS-Autoscaling] Found %d FastPods and %d ready Kubernetes pods for function %s",
		len(functionFastPods), readyPodCount, funcName)
	return len(functionFastPods), functionFastPods, readyPodCount, nil
}

// convertFastPodToHASPodInfo converts a FastPod object to a HASPodInfo object
func (r *FaSTFuncReconciler) convertFastPodToHASPodInfo(fastPod *fastpodv1.FaSTPod) (*HASPodInfo, error) {
	// Extract function name from labels
	funcName := fastPod.Labels["fast_function"]
	if funcName == "" {
		return nil, fmt.Errorf("FastPod %s does not have a function name label", fastPod.Name)
	}

	// Extract resource configuration from annotations
	nodeName := fastPod.Annotations[HASSchedNode]
	gpuID := fastPod.Annotations[HASSchedvGPU]

	// Parse SM partition
	smPartitionStr := fastPod.Annotations[fastpodv1.FaSTGShareGPUSMPartition]
	smPartition, err := strconv.ParseInt(smPartitionStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse SM partition for FastPod %s: %v", fastPod.Name, err)
	}

	// Parse quota (convert from float to int64 percentage)
	quotaStr := fastPod.Annotations[fastpodv1.FaSTGShareGPUQuotaRequest]
	quotaFloat, err := strconv.ParseFloat(quotaStr, 64)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse quota for FastPod %s: %v", fastPod.Name, err)
	}
	quota := int64(quotaFloat * 100.0) // Convert from decimal (0.xx) to percentage

	// Parse batch size
	batchSizeStr := fastPod.Annotations[HASBatchSize]
	batchSize, err := strconv.ParseInt(batchSizeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse batch size for FastPod %s: %v", fastPod.Name, err)
	}

	// Get replica count
	replica := int64(1)
	if fastPod.Spec.Replicas != nil {
		replica = int64(*fastPod.Spec.Replicas)
	}

	// Find the row index in hasGPUs if available
	rowIdx := 0
	if _, exists := hasGPUs[nodeName]; exists {
		for _, gpu := range hasGPUs[nodeName] {
			if gpu.GPUID == gpuID {
				// Find the row index for this pod
				for i, podNames := range gpu.RowPods {
					for _, podName := range podNames {
						if podName == fastPod.Name {
							rowIdx = i
							break
						}
					}
				}
				break
			}
		}
	}

	// Calculate RPS capability based on the pod's configuration
	var rpsCapability float64
	
	// Special handling for rapp functions
	if strings.Contains(strings.ToLower(funcName), "rapp") {
		// For rapp functions, use predefined values or calculate directly
		// Use default SM partition if not specified
		if smPartition == 0 {
			smPartition = RappDefaultSM
		}
		
		// Use default batch size if not specified
		if batchSize == 0 {
			batchSize = RappDefaultBatchSize
		}
		
		// Use default quota if not specified
		if quota == 0 {
			quota = RappDefaultQuota
		}
		
		// Predefined throughput calculation for rapp
		predefinedLatency := 10.0
		rpsCapability = (1000.0 / (predefinedLatency + 70)) * float64(batchSize+1) / 2
		
		klog.Info("Using predefined rapp configuration in convertFastPodToHASPodInfo",
			"function", funcName,
			"sm", smPartition,
			"batch", batchSize,
			"quota", quota,
			"rps_capability", rpsCapability)
	} else {
		// For other functions, use the regular rapp function
		rpsCapability = rapp(funcName, batchSize, smPartition, quota)
	}

	// Create and return the HASPodInfo
	hasPodInfo := &HASPodInfo{
		FuncName:    funcName,
		NodeName:    nodeName,
		GPUID:       gpuID,
		PodName:     fastPod.Name,
		SMPartition: smPartition,
		Quota:       quota,
		Batch:       batchSize,
		Replica:     replica,
		RPS:         rpsCapability,
		RowIdx:      rowIdx,
	}

	return hasPodInfo, nil
}

// convertFastPodsToHASPodInfos converts a slice of FastPod objects to a slice of HASPodInfo objects
func (r *FaSTFuncReconciler) convertFastPodsToHASPodInfos(fastPods []*fastpodv1.FaSTPod) ([]*HASPodInfo, error) {
	result := make([]*HASPodInfo, 0, len(fastPods))
	
	for _, pod := range fastPods {
		// Extract the function name from the pod
		funcName := pod.Labels["fast_function"]
		
		// Check if this is a rapp function - if so, use special handling
		isRapp := strings.Contains(strings.ToLower(funcName), "rapp")
		
		var rps float64
		if isRapp {
			// For rapp functions, calculate RPS using a predefined formula
			// This completely bypasses the profiling system
			rps = 100.0
			
			klog.Infof("Using predefined RPS calculation for rapp function: %.2f", rps)
		} else {
			// For non-rapp functions, use the existing logic to calculate RPS
			// This will go through the normal profile calculations
			// Code to calculate RPS based on pod configuration
			// ...
			hasPodInfo, err := r.convertFastPodToHASPodInfo(pod)
			if err != nil {
				klog.Warningf("[HAS-Autoscaling] Failed to convert FastPod %s to HASPodInfo: %v", pod.Name, err)
				continue
			}
			result = append(result, hasPodInfo)
			continue
		}
		
		// Create a HASPodInfo object for the rapp function
		hasPodInfo := &HASPodInfo{
			FuncName: funcName,
			NodeName: pod.Annotations[HASSchedNode],
			GPUID:    pod.Annotations[HASSchedvGPU],
			PodName:  pod.Name,
			SMPartition: int64(RappDefaultSM),
			Quota:       int64(RappDefaultQuota),
			Batch:       int64(RappDefaultBatchSize),
			Replica:     int64(1),
			RPS:         rps,
			RowIdx:      0,
		}
		
		result = append(result, hasPodInfo)
	}

	return result, nil
}

// generateRandomString generates a random string of the specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// getEfficientConfigSafely is a wrapper function for GetMostEfficientConfig that ensures "rapp" functions are never passed to the profiling system.
func (r *FaSTFuncReconciler) getEfficientConfigSafely(funcName string) (int64, int64, int) {
	if strings.Contains(strings.ToLower(funcName), "rapp") {
		return RappDefaultSM, RappDefaultQuota, int(RappDefaultBatchSize)
	}
	return GetMostEfficientConfig(funcName)
}

// // test function
// func (r *FaSTFuncReconciler) persistentReconcile() {
// 	// update every 2 seconds
// 	ticker := time.NewTicker(2 * time.Second)
// 	defer ticker.Stop()
// 	// longReqPred := ImprovedKalmanFilter(0.0, 1.0, 10.0)
// 	// longtermCnt := 0
// 	// longtermRange := 15 // the update interval = longtermRange * tickerTime
// 	tried := false
// 	for range ticker.C {
// 		ctx := context.TODO()
// 		// logger := log.FromContext(ctx)

// 		var allHASfuncs fastfuncv1.FaSTFuncList
// 		if err := r.List(ctx, &allHASfuncs); err != nil {
// 			klog.Error(err, "Failed to get FaSTFuncs.")
// 			return
// 		}
// 		// reconcile for each HAS function
// 		for _, hasfunc := range allHASfuncs.Items {

// 			//KONTON_WS
// 			funcName := hasfunc.ObjectMeta.Name
// 			// alwasy keep one instance in the cluster, scale-to-1 policy
// 			if !tried {
// 				klog.Infof("Initializing the first instance of the HASFunc %s.", funcName)
// 				tried = true
// 				config, _ := getMostEfficientConfig()
// 				config.Replicas = 1
// 				config.NodeID = "kgpu1"
// 				config.vGPUID = nodeGPUMap[config.NodeID][0].vGPUID
// 				klog.Infof("konton_test### config: %s-%s-%d.", config.NodeID, config.vGPUID, config.BatchSize)
// 				var configslist []*FaSTPodConfig
// 				configslist = append(configslist, &config)
// 				fastpods, _ := r.configs2FaSTPods(&hasfunc, configslist)
// 				err := r.Create(context.TODO(), fastpods[0])
// 				if err != nil {
// 					klog.Errorf("Failed to create the fastfunc %s.", funcName)
// 				}
// 			}

// 			// retrive the past T(T0) second's rps
// 			klog.Infof("Checking HASFunc %s.", funcName)
// 			// // --------------- KONTON_TEST_START --------------------
// 			pstRPST0 := float64(0.0)
// 			// qItv := 30
// 			// queryT0 := fmt.Sprintf("rate(gateway_function_invocation_total{function_name='%s.%s'}[15s])", funcName, hasfunc.ObjectMeta.Namespace)
// 			// klog.Infof("Prometheus Query: %s.", queryT0)
// 			// queryT0 := fmt.Sprintf(`sum((http_requests_total{path="/function/fastfunc-resnet/predict"}[30s])`)
// 			// queryResT0, _, err := r.promv1api.Query(ctx, queryT0, time.Now())

// 			queryResT0, _, err := r.promv1api.Query(ctx, `sum(http_requests_total{path=~"/function/fastfunc-resnet.*"})`, time.Now())

// 			if err != nil {
// 				klog.Errorf("Error Failed to get RPS of function %s: %s.", funcName, err.Error())
// 				continue
// 			}
// 			if queryResT0.(model.Vector).Len() != 0 {
// 				klog.Infof("The past rps vec for function %s is %v.", funcName, queryResT0)
// 				pstRPST0 = float64(queryResT0.(model.Vector)[0].Value)
// 			}
// 			klog.Infof("The past [%ds], rps for function %s is %f.", 8, funcName, pstRPST0)
// 			// ------------  KONTON_TEST_END   ------------------------
// 		}
// 	}
// }
