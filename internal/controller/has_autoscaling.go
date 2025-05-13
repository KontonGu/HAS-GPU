package controller

import (
	//"fmt"
	"sort"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

const (
	ScaleUpThredshold   = 0.9
	ScaleDownThredshold = 0.30
	DeltaIq             = int(10)
	DeltaIs             = int(20)
	HGOThredshold       = 0.95
	GPUSMConfigNum      = 2
	// Cooldown period in seconds
	ScaleDownCooldownPeriod = 10 // 20 seconds cooldown for scale-down operations
	MinQuota                = int64(10)
	
	// Default configuration for rapp model
	RappDefaultSM        = int64(10)  // Default SM partition for rapp model
	RappDefaultQuota     = int64(20)  // Default quota for rapp model
	RappDefaultBatchSize = int64(2)   // Default batch size for rapp model
	RappMinRPS           = float64(20) // Minimum RPS threshold for rapp model
)

// Map to track the last scale-down time for each function
var lastScaleDownTime = make(map[string]time.Time)

// Horizontal and Vertical Scaling Policy for a Function
func (r *FaSTFuncReconciler) HASFuncAutoScale(predRPS float64, podsinfo []*HASPodInfo, hasgpus map[string][]HASGPUInfo) []HASAction {
	// Check if podsinfo is empty
	if len(podsinfo) == 0 {
		klog.Info("HASFuncAutoScale: No pods info provided, skipping autoscaling")
		return nil
	}

	// Get function name from the first pod (all pods should be for the same function)
	funcName := podsinfo[0].FuncName
	
	// Skip all processing for rapp functions
	if strings.Contains(strings.ToLower(funcName), "rapp") {
		klog.Info("HASFuncAutoScale: Skipping autoscaling for rapp function")
		return nil
	}

	// Verify that there is at least one FastPod and one ready pod for this function
	podCount, _, readyPodCount, err := r.checkFaSTPodsForFunction(funcName)
	if err != nil {
		klog.Errorf("HASFuncAutoScale: Error checking FastPods for function %s: %v", funcName, err)
		return nil
	}

	// Skip autoscaling if no FastPods are found or if no pods are ready
	if podCount == 0 {
		klog.Infof("HASFuncAutoScale: No actual FastPods found for function %s. Skipping autoscaling.", funcName)
		return nil
	}

	if readyPodCount == 0 {
		klog.Infof("HASFuncAutoScale: No ready pods found for function %s. Skipping autoscaling.", funcName)
		return nil
	}

	klog.Infof("HASFuncAutoScale: Found %d FastPods (%d ready) for function %s", podCount, readyPodCount, funcName)

	// judge if autoscaling necessary
	totalRPS := r.GetFuncRPSCapability(podsinfo)
	if predRPS > ScaleUpThredshold*totalRPS {
		deltaRPS := predRPS - ScaleUpThredshold*totalRPS
		klog.Infof("HASFuncAutoScale: Scale-up triggered for %s. Additional capacity needed: %.2f RPS", funcName, deltaRPS)
		actions := r.HASFuncScaleUp(deltaRPS, podsinfo, hasgpus)
		klog.Infof("HASFuncScaleUp: Scale-up complete for %s. Generated %d scaling actions", funcName, len(actions))
		return actions
	} else if predRPS < ScaleDownThredshold*totalRPS {
		// Check if we're in the cooldown period for scale-down
		lastScaleDown, exists := lastScaleDownTime[funcName]
		now := time.Now()

		if exists && now.Sub(lastScaleDown).Seconds() < ScaleDownCooldownPeriod {
			// Still in cooldown period, skip scale-down
			klog.Infof("HASFuncAutoScale: Scale-down for %s skipped due to cooldown period. Time since last scale-down: %.2f seconds (cooldown: %.2f seconds)",
				funcName, now.Sub(lastScaleDown).Seconds(), ScaleDownCooldownPeriod)
			return nil
		}

		deltaRPS := ScaleUpThredshold*totalRPS - predRPS
		klog.Infof("HASFuncAutoScale: Scale-down triggered for %s. Excess capacity: %.2f RPS", funcName, deltaRPS)
		actions := r.HASFuncScaleDown(deltaRPS, podsinfo, hasgpus)

		// If we're actually scaling down (actions returned), update the last scale-down time
		if len(actions) > 0 {
			lastScaleDownTime[funcName] = now
			klog.Infof("HASFuncAutoScale: Scale-down complete for %s. Generated %d scaling actions. Cooldown period started",
				funcName, len(actions))
		} else {
			klog.Infof("HASFuncAutoScale: No scale-down actions generated for %s despite being under threshold", funcName)
		}

		return actions
	}
	klog.Infof("HASFuncAutoScale: No scaling needed for %s. Current load is within thresholds", funcName)
	return nil
}

/*
Scaling-up (vertical + horizontal scaling) for a function.
Parameters:
  - deltaRPS:    rps deficit of the function;
  - podsinfo:    currently running pods for the function
  - HASGPUInfo:  gpu occpuancy status of all gpus in the cluster

Returns:
  - HASAction:   the list of actions required (types including scaling-up instance (1), vertical-scaling (0), scaling-down(-1)).
*/
func (r *FaSTFuncReconciler) HASFuncScaleUp(deltaRPS float64, podsinfo []*HASPodInfo, hasgpus map[string][]HASGPUInfo) []HASAction {
	// sort the pods running of the HAS function by the sm value, higher sm first
	sort.Slice(podsinfo, func(i, j int) bool {
		return podsinfo[i].SMPartition > podsinfo[j].SMPartition
	})

	klog.Infof("HASFuncScaleUp: Sorted pods by SMPartition (descending)")

	actions := []HASAction{}
	// tranverse the sorted pods to do the vertical autoscaling
	deltaR := deltaRPS
	for _, pod := range podsinfo {
		gpuid := pod.GPUID
		sm := pod.SMPartition
		quota := pod.Quota
		batch := pod.Batch
		// the maximum avaialble Quota in the GPU (gpuid) of the pod
		Aq, gnodename, gidx := r.RetriveMaxAvailQuotaForPod(pod.PodName, gpuid, hasgpus) // available quota

		klog.Infof("HASFuncScaleUp: Available quota for pod %s: %d on node %s (GPU idx: %d)",
			pod.PodName, Aq, gnodename, gidx)

		n := int(1) // number of stepsize
		deltaC := float64(0.0)
		addQuota := DeltaIq * n
		newQuota := quota

		klog.Infof("HASFuncScaleUp: Looking for optimal quota increase. Initial addQuota: %d", addQuota)

		for addQuota <= int(Aq) && deltaC < deltaR {
			newQuota = quota + int64(addQuota)
			podCap := rapp(pod.FuncName, batch, sm, newQuota)
			deltaC = podCap - pod.RPS
			n += 1
			addQuota = DeltaIq * n
		}
		deltaR -= deltaC
		klog.Infof("HASFuncScaleUp: After vertica l scaling pod %s: newQuota=%d, capacity increase		=%.2f, remaining deltaRPS=%.2f",
			pod.PodName, newQuota, deltaC, deltaR)

		//		 update pod information
		pod.Quota = newQuota
		pod.RPS = pod.RPS + deltaC
		act := HASAction{ActionType: 0, PodItem: pod}
		actions = append(actions, act)

		// update gpu informations related to the pod
		cfg := hasgpus[gnodename][gidx].PodsConfig[pod.PodName]
		oldHGO := hasgpus[gnodename][gidx].HGO
		hasgpus[gnodename][gidx].HGO += float64((cfg.SMPartition / 100.0) * (newQuota - cfg.Quota) / 100.0)

		klog.Infof("HASFuncScaleUp: Updated GPU %s on node %s HGO: %.4f -> %.4f",
			gpuid, gnodename, oldHGO, hasgpus[gnodename][gidx].HGO)

		cfg.Quota = newQuota
		hasgpus[gnodename][gidx].PodsConfig[pod.PodName] = cfg
	}

	// horizontal scaling if the vertical scaling still not satisfies the deltaR
	var prevDeltaR float64
	var newGPUs map[string][]string
	for deltaR > 0 {
		// Check if the function is "rapp" - we don't want to scale rapp
		funcType := r.getFunctionTypeFromName(podsinfo[0].FuncName)
		if funcType == "rapp" {
			klog.Infof("HASFuncScaleUp: Skipping horizontal scaling for rapp model")
			break
		}

		// used GPU first
		usedGPUs := RetrieveUsedGPUOrderedByHGO(hasgpus, HGOThredshold) // ordered by HGO (HAS GPU Occupancy)
		// To evaluate if one pod instance can satisfy the deltaR (if not, skip and create the pod in a new GPU)
		klog.Infof("HASFuncScaleUp: Starting horizontal scaling with deltaR=%.2f", deltaR)
		for _, gpuinfo := range usedGPUs {
			smConfigs, reQuota := r.RetriveMaxAvailQuotaForGPU(gpuinfo.NodeName, gpuinfo.GPUID, hasgpus) //rowIdxes <--> smConfigs <-->  reQuota

			// Find the best SM configuration and quota for the new pod
			var bestSMIdx int = -1
			var maxCap float64 = 0.0
			var bestQuota int64 = 0

			// Loop through available SM configurations to find the best one
			for idx, sm := range smConfigs {
				if sm <= 0 {
					klog.Infof("HASFuncScaleUp: SM config[%d] has invalid or uninitialized SM value %d",
						idx, sm)
					// Instead of skipping, use a default SM value for uninitialized rows
					if reQuota[idx] > 0 {
						// Get the function name from the first pod in the function
						var funcName string
						for _, pod := range podsinfo {
							funcName = pod.FuncName
							break
						}

						// Calculate optimal SM partition based on function name and required throughput
						optimalSM := r.getOptimalSMPartition(funcName, deltaR)
						sm = int64(optimalSM)
						klog.Infof("HASFuncScaleUp: Using calculated optimal SM value of %d for function %s with RPS %f on row %d",
							sm, funcName, deltaR, idx)

						// Update the actual SM configuration in the GPU
						updateSMConfigForGPU(hasgpus, gpuinfo.NodeName, gpuinfo.GPUID, idx, sm)
					} else {
						klog.Infof("HASFuncScaleUp: Skipping SM config[%d] with no quota available", idx)
						continue
					}
				}

				if reQuota[idx] <= 0 {
					klog.Infof("HASFuncScaleUp: Skipping SM config[%d] with no remaining quota (SM=%d, quota=%d)",
						idx, sm, reQuota[idx])
					continue
				}

				// Calculate maximum possible quota for this SM configuration
				maxPossibleQuota := reQuota[idx]

				// Try different quota values in steps of DeltaIq
				n := int(1)
				addQuota := DeltaIq * n

				for addQuota <= int(maxPossibleQuota) {
					quota := int64(addQuota)
					// Calculate capacity with this configuration
					// Get the function name from the first pod in the function
					var funcName string
					if len(podsinfo) > 0 {
						funcName = podsinfo[0].FuncName
					}
					podCap := rapp(funcName, int64(0), sm, quota) // Pass the actual function name instead of empty string

					// If this configuration gives better capac		ity, remember it
					if podCap > maxCap {
						maxCap = podCap
						bestSMIdx = idx
						bestQuota = quota
					}

					n += 1
					addQuota = DeltaIq * n
				}
			}

			// If we found a valid configuration and it can handle the remaining deltaR
			if bestSMIdx >= 0 && maxCap > 0 {
				// Create a new pod with the best configuration
				sm := smConfigs[bestSMIdx]
				quota := bestQuota

				// Generate a unique pod name
				podName := podsinfo[0].FuncName + getResKeyName(quota, sm)

				// Find the optimal row index for the new pod
				rowIdx := FindOptimalRowIdx(gpuinfo, sm)

				// Create new pod info
				newPod := &HASPodInfo{
					FuncName:    podsinfo[0].FuncName,
					NodeName:    gpuinfo.NodeName,
					GPUID:       gpuinfo.GPUID,
					PodName:     podName,
					SMPartition: sm,
					Quota:       quota,
					Batch:       podsinfo[0].Batch, // Use the same batch size as existing pods
					Replica:     int64(1),
					RPS:         maxCap,
					RowIdx:      rowIdx,
				}

				// Add the pod to the GPU
				r.addPodtoHASGPU(newPod, hasgpus)

				// Create action for the new pod
				act := HASAction{ActionType: 1, PodItem: newPod}
				actions = append(actions, act)

				// Update remaining deltaR
				deltaR -= maxCap

				// If we've satisfied the deltaR requirement, break out of the loop
				if deltaR <= 0 {
					break
				}
			}
		}

		// If we still need more capacity and there are no suitable existing GPUs,
		// try to allocate on a new GPU
		if deltaR > 0 {
			// Get unused GPUs
			newGPUs = RetrieveNewGPU(hasgpus)
			if len(newGPUs) > 0 {
				// Find the most efficient configuration for the remaining deltaR
				// This would typically involve finding the optimal batch size, SM partition, and quota
				// for the specific function and the available GPU

				// For simplicity, we'll use a helper function that should be implemented
				// to find the most efficient configuration
				var nodeName string
				var gpuID string

				// Get the first available GPU
				for node, gpus := range newGPUs {
					if len(gpus) > 0 {
						nodeName = node
						gpuID = gpus[0]
						break
					}
				}

				if nodeName != "" && gpuID != "" {
					// Get most efficient configuration for the remaining deltaR
					// This should be modified to take deltaR as input
					sm, quota, batchSize := GetMostEfficientConfig() // This should be modified to take deltaR as input

					// Create a FaSTPodConfig to use with the values
					bestConfig := FaSTPodConfig{
						SMPartition: sm,
						Quota:       quota,
						BatchSize:   batchSize,
					}

					// Generate a unique pod name
					podName := podsinfo[0].FuncName + getResKeyName(bestConfig.Quota, bestConfig.SMPartition) + generateRandomString(4)

					// Find the optimal row index for the new pod
					var gpuInfo HASGPUInfo
					for _, gpu := range hasgpus[nodeName] {
						if gpu.GPUID == gpuID {
							gpuInfo = gpu
							break
						}
					}
					rowIdx := FindOptimalRowIdx(gpuInfo, bestConfig.SMPartition)

					// Create new pod info
					newPod := &HASPodInfo{
						FuncName:    podsinfo[0].FuncName,
						NodeName:    nodeName,
						GPUID:       gpuID,
						PodName:     podName,
						SMPartition: bestConfig.SMPartition,
						Quota:       bestConfig.Quota,
						Batch:       int64(bestConfig.BatchSize),
						Replica:     int64(1),
						RPS:         rapp(podsinfo[0].FuncName, int64(bestConfig.BatchSize), bestConfig.SMPartition, bestConfig.Quota),
						RowIdx:      rowIdx,
					}

					// Add the pod to the GPU
					r.addPodtoHASGPU(newPod, hasgpus)

					// Create action for the new pod
					act := HASAction{ActionType: 1, PodItem: newPod}
					actions = append(actions, act)

					// Update remaining deltaR
					deltaR -= newPod.RPS
				}
			}
		}
		// Safety check to prevent infinite loop if no progress is made
		if len(newGPUs) == 0 || (len(usedGPUs) == 0 && len(newGPUs) == 0) {
			// If no GPUs are available under normal thresholds, but we still need to scale,
			// try to use any existing GPU regardless of HGO threshold as a fallback
			if deltaR > 0 {
				klog.Infof("HASFuncScaleUp: No suitable GPUs found under threshold. Trying fallback to use any existing GPU.")

				// Get all GPUs regardless of HGO threshold
				allGPUs := make([]HASGPUInfo, 0)
				for nodeName, nodegpus := range hasgpus {
					for _, gpu := range nodegpus {
						allGPUs = append(allGPUs, gpu)
						klog.Infof("HASFuncScaleUp: Found GPU %s on node %s with HGO=%.4f",
							gpu.GPUID, nodeName, gpu.HGO)
					}
				}

				klog.Infof("HASFuncScaleUp: Found %d total GPUs in the cluster", len(allGPUs))

				// Sort by HGO to use the least occupied first
				sort.Slice(allGPUs, func(i, j int) bool {
					return allGPUs[i].HGO < allGPUs[j].HGO
				})

				if len(allGPUs) > 0 {
					// Try to use the first available GPU
					gpuinfo := allGPUs[0]
					klog.Infof("HASFuncScaleUp: Attempting to use GPU %s on node %s with HGO=%.4f as fallback",
						gpuinfo.GPUID, gpuinfo.NodeName, gpuinfo.HGO)

					// Debug the GPU structure
					klog.Infof("HASFuncScaleUp: GPU structure - SMConfigs: %v", gpuinfo.SMConfigs)
					klog.Infof("HASFuncScaleUp: GPU structure - RowPods: %v", gpuinfo.RowPods)

					smConfigs, reQuota := r.RetriveMaxAvailQuotaForGPU(gpuinfo.NodeName, gpuinfo.GPUID, hasgpus)

					klog.Infof("HASFuncScaleUp: Retrieved %d SM configurations and %d quota values for fallback GPU",
						len(smConfigs), len(reQuota))

					// Log available configurations
					foundViableConfig := false
					for idx, sm := range smConfigs {
						remainingQuota := int64(0)
						if idx < len(reQuota) {
							remainingQuota = reQuota[idx]
						}

						klog.Infof("HASFuncScaleUp: SM config[%d]: SM=%d, remaining quota=%d",
							idx, sm, remainingQuota)

						if sm > 0 && remainingQuota > 0 {
							foundViableConfig = true
						}
					}

					if !foundViableConfig {
						klog.Warning("HASFuncScaleUp: No viable SM configurations found on fallback GPU")
					}

					// Find any viable configuration
					for idx, sm := range smConfigs {
						if idx >= len(reQuota) {
							klog.Warningf("HASFuncScaleUp: Index %d out of bounds for reQuota (len=%d)",
								idx, len(reQuota))
							continue
						}

						if sm <= 0 {
							klog.Infof("HASFuncScaleUp: SM config[%d] has invalid or uninitialized SM value %d",
								idx, sm)
							// Instead of skipping, use a default SM value for uninitialized rows
							if reQuota[idx] > 0 {
								// Update the actual SM configuration in the GPU
								defaultSM := int64(50)
								updateSMConfigForGPU(hasgpus, gpuinfo.NodeName, gpuinfo.GPUID, idx, defaultSM)
								sm = defaultSM
								klog.Infof("HASFuncScaleUp: Using default SM value of %d for row %d", defaultSM, idx)
							} else {
								klog.Infof("HASFuncScaleUp: Skipping SM config[%d] with no quota available", idx)
								continue
							}
						}

						if reQuota[idx] <= 0 {
							klog.Infof("HASFuncScaleUp: Skipping SM config[%d] with no remaining quota (SM=%d, quota=%d)",
								idx, sm, reQuota[idx])
							continue
						}

						// Create a new pod with minimal viable configuration
						quota := MinQuota
						if reQuota[idx] < MinQuota {
							quota = reQuota[idx]
						}

						podName := podsinfo[0].FuncName + getResKeyName(quota, sm)
						rowIdx := FindOptimalRowIdx(gpuinfo, sm)

						klog.Infof("HASFuncScaleUp: Found viable configuration: SM=%d, quota=%d, rowIdx=%d",
							sm, quota, rowIdx)

						newPod := &HASPodInfo{
							FuncName:    podsinfo[0].FuncName,
							NodeName:    gpuinfo.NodeName,
							GPUID:       gpuinfo.GPUID,
							PodName:     podName,
							SMPartition: sm,
							Quota:       quota,
							Batch:       podsinfo[0].Batch,
							Replica:     int64(1),
							RPS:         rapp(podsinfo[0].FuncName, podsinfo[0].Batch, sm, quota),
							RowIdx:      rowIdx,
						}

						// Add the pod to the GPU
						r.addPodtoHASGPU(newPod, hasgpus)

						// Create action for the new pod
						act := HASAction{ActionType: 1, PodItem: newPod}
						actions = append(actions, act)

						// Update remaining deltaR
						deltaR -= newPod.RPS
						klog.Infof("HASFuncScaleUp: Created fallback pod %s on existing GPU %s with SM=%d, quota=%d, RPS=%.2f",
							podName, gpuinfo.GPUID, sm, quota, newPod.RPS)
						break
					}
				} else {
					klog.Warning("HASFuncScaleUp: No GPUs found in the cluster")
				}
			}

			klog.Warning("HASFuncScaleUp: No suitable GPUs found and no progress made, breaking loop")
			break
		}
		// Additional safety check: if deltaR hasn't changed in this iteration, break to avoid infinite loop
		if prevDeltaR == deltaR && deltaR > 0 {
			klog.Warning("HASFuncScaleUp: No progress made in reducing deltaRPS (%.2f), breaking loop", deltaR)
			break
		}
		prevDeltaR = deltaR
	}

	klog.Infof("HASFuncScaleUp: Completed with %d actions. Final remaining deltaRPS: %.2f", len(actions), deltaR)
	return actions
}

func (r *FaSTFuncReconciler) HASFuncScaleDown(deltaRPS float64, podsinfo []*HASPodInfo, hasgpus map[string][]HASGPUInfo) []HASAction {
	// Sort pods by SM partition, fewer SMs first (opposite of scale-up)
	sort.Slice(podsinfo, func(i, j int) bool {
		return podsinfo[i].SMPartition < podsinfo[j].SMPartition
	})

	actions := []HASAction{}
	deltaR := deltaRPS

	// Define minimum RPS threshold to keep a function running
	minRPS := float64(0.1) // Minimum RPS to keep a function running

	// Track GPUs that might become empty after scaling down
	emptyGPUs := make(map[string]map[string]bool) // map[nodeName][gpuID]

	// First pass: reduce quota for pods
	for _, pod := range podsinfo {
		// Skip if we've already reduced enough capacity
		if deltaR <= 0 {
			break
		}

		// Calculate how much RPS we can reduce from this pod
		currentRPS := pod.RPS
		gpuid := pod.GPUID
		sm := pod.SMPartition
		quota := pod.Quota
		nodeName := pod.NodeName

		// Initialize node in emptyGPUs map if not exists
		if _, exists := emptyGPUs[nodeName]; !exists {
			emptyGPUs[nodeName] = make(map[string]bool)
		}

		// Try reducing quota in steps
		n := int(1)
		deltaC := float64(0.0)
		reduceQuota := DeltaIq * n
		newQuota := quota

		for int(quota)-reduceQuota > 0 && deltaC < deltaR {
			newQuota = quota - int64(reduceQuota)
			podCap := rapp(pod.FuncName, pod.Batch, sm, newQuota)
			deltaC = currentRPS - podCap

			// Don't reduce below minimum RPS for the last pod of a function
			if len(podsinfo) == 1 && podCap < minRPS {
				break
			}

			n += 1
			reduceQuota = DeltaIq * n
		}

		// If we can reduce quota
		if newQuota < quota {
			deltaR -= deltaC

			// Update pod information
			pod.Quota = newQuota
			pod.RPS = pod.RPS - deltaC

			// Create scale-down action
			act := HASAction{ActionType: 0, PodItem: pod}
			actions = append(actions, act)

			// Update GPU information
			if _, exists := hasgpus[nodeName]; exists {
				for idx, gpu := range hasgpus[nodeName] {
					if gpu.GPUID == gpuid {
						cfg := hasgpus[nodeName][idx].PodsConfig[pod.PodName]
						hasgpus[nodeName][idx].HGO -= float64((cfg.SMPartition / 100.0) * (cfg.Quota - newQuota) / 100.0)
						cfg.Quota = newQuota
						hasgpus[nodeName][idx].PodsConfig[pod.PodName] = cfg
						break
					}
				}
			}

			// If quota is reduced to minimum, mark for potential removal
			if newQuota <= int64(DeltaIq) {
				emptyGPUs[nodeName][gpuid] = true
			}
		}
	}

	// Second pass: remove pods if necessary (horizontal scale-down)
	if deltaR > 0 {
		for i := 0; i < len(podsinfo); i++ {
			// Skip if we've already reduced enough capacity or this is the last pod
			if deltaR <= 0 || len(podsinfo) <= 1 {
				break
			}

			pod := podsinfo[i]

			// Skip pods with high SM partition as they're more efficient
			if pod.SMPartition > int64(50) { // Arbitrary threshold
				continue
			}

			// Calculate RPS contribution of this pod
			podRPS := pod.RPS

			// If removing this pod would satisfy our scale-down requirement
			if podRPS <= deltaR {
				deltaR -= podRPS

				// Create delete action
				act := HASAction{ActionType: -1, PodItem: pod}
				actions = append(actions, act)

				// Update GPU information - remove pod from GPU
				nodeName := pod.NodeName
				gpuid := pod.GPUID

				if _, exists := hasgpus[nodeName]; exists {
					for idx, gpu := range hasgpus[nodeName] {
						if gpu.GPUID == gpuid {
							// Find the row index of this pod
							rowIdx := -1
							for ridx, row := range gpu.RowPods {
								for _, podName := range row {
									if podName == pod.PodName {
										rowIdx = ridx
										break
									}
								}
								if rowIdx >= 0 {
									break
								}
							}

							// Remove pod from row
							if rowIdx >= 0 {
								newRow := []string{}
								for _, podName := range gpu.RowPods[rowIdx] {
									if podName != pod.PodName {
										newRow = append(newRow, podName)
									}
								}

								// Update row
								hasgpus[nodeName][idx].RowPods[rowIdx] = newRow

								// Update HGO
								cfg := hasgpus[nodeName][idx].PodsConfig[pod.PodName]
								hasgpus[nodeName][idx].HGO -= float64((cfg.SMPartition / 100.0) * (cfg.Quota) / 100.0)

								// Remove pod from configs
								delete(hasgpus[nodeName][idx].PodsConfig, pod.PodName)

								// If row is empty, mark GPU as potentially unused
								if len(newRow) == 0 {
									emptyGPUs[nodeName][gpuid] = true
								}
							}
							break
						}
					}
				}

				// Remove pod from podsinfo (adjust loop counter)
				podsinfo = append(podsinfo[:i], podsinfo[i+1:]...)
				i-- // Adjust loop counter
			}
		}
	}

	// Clean up empty GPUs
	for nodeName, gpus := range emptyGPUs {
		for gpuid := range gpus {
			isGPUEmpty := true

			// Check if GPU is actually empty
			if _, exists := hasgpus[nodeName]; exists {
				for idx, gpu := range hasgpus[nodeName] {
					if gpu.GPUID == gpuid {
						// Check if all rows are empty
						for _, row := range gpu.RowPods {
							if len(row) > 0 {
								isGPUEmpty = false
								break
							}
						}

						// If GPU is empty, remove it from hasGPUs
						if isGPUEmpty {
							hasgpus[nodeName] = append(hasgpus[nodeName][:idx], hasgpus[nodeName][idx+1:]...)
						}
						break
					}
				}
			}
		}
	}

	return actions
}

func (r *FaSTFuncReconciler) addPodtoHASGPU(podinfo *HASPodInfo, hasgpus map[string][]HASGPUInfo) {
	// Get the node name and GPU ID from the pod info
	nodeName := podinfo.NodeName
	gpuID := podinfo.GPUID
	rowIdx := podinfo.RowIdx

	// Find the GPU in the hasgpus map
	var gpuIndex int = -1

	// Check if node exists in the map, if not initialize it
	if _, exists := hasgpus[nodeName]; !exists {
		hasgpus[nodeName] = []HASGPUInfo{}
	}

	// Find the GPU in the node's GPU list
	for idx, gpu := range hasgpus[nodeName] {
		if gpu.GPUID == gpuID {
			gpuIndex = idx
			break
		}
	}

	// If GPU not found, create a new HASGPUInfo for it
	if gpuIndex == -1 {
		newGPU := HASGPUInfo{
			NodeName:   nodeName,
			GPUID:      gpuID,
			SMConfigs:  make([]int64, GPUSMConfigNum),
			HGO:        0.0,
			RowPods:    make([][]string, GPUSMConfigNum),
			PodsConfig: make(map[string]HASResConfig),
		}

		// Initialize SM configurations (default to -1 meaning no pod running)
		for i := range newGPU.SMConfigs {
			newGPU.SMConfigs[i] = -1
			newGPU.RowPods[i] = []string{}
		}

		// Add the new GPU to the node
		hasgpus[nodeName] = append(hasgpus[nodeName], newGPU)
		gpuIndex = len(hasgpus[nodeName]) - 1
	}

	// Double-check that the node and GPU index are valid before accessing
	if len(hasgpus[nodeName]) == 0 || gpuIndex < 0 || gpuIndex >= len(hasgpus[nodeName]) {
		klog.Errorf("Invalid GPU index %d for node %s in addPodtoHASGPU", gpuIndex, nodeName)
		return
	}

	// Validate rowIdx is within bounds
	if rowIdx < 0 || rowIdx >= GPUSMConfigNum {
		klog.Errorf("Invalid row index %d for node %s, GPU %s in addPodtoHASGPU", rowIdx, nodeName, gpuID)
		return
	}

	// Initialize arrays if they're nil or empty
	if hasgpus[nodeName][gpuIndex].SMConfigs == nil || len(hasgpus[nodeName][gpuIndex].SMConfigs) == 0 {
		hasgpus[nodeName][gpuIndex].SMConfigs = make([]int64, GPUSMConfigNum)
		for i := range hasgpus[nodeName][gpuIndex].SMConfigs {
			hasgpus[nodeName][gpuIndex].SMConfigs[i] = -1
		}
	}

	if hasgpus[nodeName][gpuIndex].RowPods == nil || len(hasgpus[nodeName][gpuIndex].RowPods) == 0 {
		hasgpus[nodeName][gpuIndex].RowPods = make([][]string, GPUSMConfigNum)
		for i := range hasgpus[nodeName][gpuIndex].RowPods {
			hasgpus[nodeName][gpuIndex].RowPods[i] = []string{}
		}
	}

	// Ensure PodsConfig map is initialized
	if hasgpus[nodeName][gpuIndex].PodsConfig == nil {
		hasgpus[nodeName][gpuIndex].PodsConfig = make(map[string]HASResConfig)
	}

	// Set the SM configuration for the row if it's not already set
	if hasgpus[nodeName][gpuIndex].SMConfigs[rowIdx] == -1 {
		// Ensure SM partition is valid (positive)
		if podinfo.SMPartition <= 0 {
			klog.Warningf("Invalid SM partition value %d for pod %s, using default value 50",
				podinfo.SMPartition, podinfo.PodName)
			podinfo.SMPartition = 50 // Use default value if invalid
		}
		hasgpus[nodeName][gpuIndex].SMConfigs[rowIdx] = podinfo.SMPartition
	} else if podinfo.SMPartition <= 0 {
		// If the pod has an invalid partition but the row already has a valid one,
		// use the row's existing partition value
		klog.Warningf("Pod %s has invalid partition %d, using row's existing partition %d instead",
			podinfo.PodName, podinfo.SMPartition, hasgpus[nodeName][gpuIndex].SMConfigs[rowIdx])
		podinfo.SMPartition = hasgpus[nodeName][gpuIndex].SMConfigs[rowIdx]
	} else if podinfo.SMPartition != hasgpus[nodeName][gpuIndex].SMConfigs[rowIdx] {
		// If the pod has a different partition than the row, log a warning
		klog.Warningf("Pod %s has partition %d but row %d already has partition %d, using row's existing partition",
			podinfo.PodName, podinfo.SMPartition, rowIdx, hasgpus[nodeName][gpuIndex].SMConfigs[rowIdx])
		podinfo.SMPartition = hasgpus[nodeName][gpuIndex].SMConfigs[rowIdx]
	}

	// Add pod to the row
	hasgpus[nodeName][gpuIndex].RowPods[rowIdx] = append(hasgpus[nodeName][gpuIndex].RowPods[rowIdx], podinfo.PodName)

	// Add pod configuration
	hasgpus[nodeName][gpuIndex].PodsConfig[podinfo.PodName] = HASResConfig{
		Name:        podinfo.PodName,
		SMPartition: podinfo.SMPartition,
		Quota:       podinfo.Quota,
		Memory:      podinfo.Memory,
	}

	// Update HGO (HAS GPU Occupancy)
	hasgpus[nodeName][gpuIndex].HGO += float64((podinfo.SMPartition / 100.0) * (podinfo.Quota) / 100.0)
}

// Helper function to update SM configuration for uninitialized rows
func updateSMConfigForGPU(hasgpus map[string][]HASGPUInfo, nodeName string, gpuID string, rowIdx int, newSM int64) bool {
	// Find the GPU in the hasgpus map
	for nodeIdx, gpus := range hasgpus {
		if nodeIdx != nodeName {
			continue
		}

		for gpuIdx, gpu := range gpus {
			if gpu.GPUID == gpuID {
				// Only update if it's uninitialized
				if hasgpus[nodeIdx][gpuIdx].SMConfigs[rowIdx] <= 0 {
					klog.Infof("Updating SM config for GPU %s on node %s, row %d from %d to %d",
						gpuID, nodeName, rowIdx, hasgpus[nodeIdx][gpuIdx].SMConfigs[rowIdx], newSM)
					hasgpus[nodeIdx][gpuIdx].SMConfigs[rowIdx] = newSM
					return true
				}
				return false
			}
		}
	}
	return false
}

// Get the maximum remaining quota in the GPU of the pod <podname>
// Parameters:
//   - hasgpus:    cluster GPU information
//   - podname:    the name of the pod to retrive
//
// Returns:
//   - int64:     remaining quota in the pod's gpu.
//   - string:    The name of the node where the pod is located.
//   - int:       The RowIdx of the pod in the GPU.
func (r *FaSTFuncReconciler) RetriveMaxAvailQuotaForPod(podname string, gpuid string, hasgpus map[string][]HASGPUInfo) (int64, string, int) {
	var podgpu HASGPUInfo
	var nodeName string
	var gpuIdx int
	// get HASGPUInfo of gpuid
	for nodename, nodegpus := range hasgpus {
		for idx, gpu := range nodegpus {
			if gpu.GPUID == gpuid {
				podgpu = gpu
				nodeName = nodename
				gpuIdx = idx
			}
		}
	}
	// retrieve the resource row of the pod
	var rowIdx int // the sm configuration idx of the pod in the gpu
	for idx, row := range podgpu.RowPods {
		for _, tpodname := range row {
			if tpodname == podname {
				rowIdx = idx
			}
		}
	}
	reQuota := int64(100)
	// traverse the row to get remaining quota
	for _, tpodname := range podgpu.RowPods[rowIdx] {
		reQuota -= podgpu.PodsConfig[tpodname].Quota
	}
	return reQuota, nodeName, gpuIdx
}

// Get the maximum remaining quota in the GPU of the pod <podname>
// Parameters:
//   - hasgpus:      cluster GPU information
//   - nodename:     The node of the gpu to retrieve.
//   - gpuid:        The gpu to retrieve.
//
// Returns:
//   - []int64:    SM configuration of the rows.
//   - []int64:    Remaining quota of the rows.    1-in-1 corresponds
func (r *FaSTFuncReconciler) RetriveMaxAvailQuotaForGPU(nodename string, gpuid string, hasgpus map[string][]HASGPUInfo) ([]int64, []int64) {
	var tmp HASGPUInfo
	for _, gpuinfo := range hasgpus[nodename] {
		if gpuinfo.GPUID == gpuid {
			tmp = gpuinfo
		}
	}
	reQuota := make([]int64, 0)
	smConfigs := make([]int64, 0)
	for idx, pods := range tmp.RowPods {
		tquota := int64(0)
		for _, pod := range pods {
			tquota += tmp.PodsConfig[pod].Quota
		}
		smConfigs = append(smConfigs, tmp.SMConfigs[idx])
		reQuota = append(reQuota, 100-tquota)
	}

	return smConfigs, reQuota
}

func (r *FaSTFuncReconciler) GetFuncRPSCapability(PodsInfo []*HASPodInfo) float64 {
	var totalRPS float64 = 0.0
	for _, pod := range PodsInfo {
		totalRPS += pod.RPS
	}
	return totalRPS
}

// Retrieve the used GPU but with HGO (HAS GPU Occupancy) less than maxhgo.
// Parameters:
//   - hasgpus:    cluster GPU information
//   - maxhgo:     maximum HGO limit
//
// Returns:
//   - map[string][]string:   The nodes and gpus in the nodes.
func RetrieveUsedGPU(hasgpus map[string][]HASGPUInfo, maxhgo float64) map[string][]string {
	availGPUs := make(map[string][]string)
	for nodename, nodegpus := range hasgpus {
		for _, gpu := range nodegpus {
			if gpu.HGO < float64(maxhgo) {
				availGPUs[nodename] = append(availGPUs[nodename], gpu.GPUID)
			}
		}
	}
	return availGPUs
}

// Retrieve the used GPU but with HGO (HAS GPU Occupancy) less than maxhgo.
// Parameters:
//   - hasgpus:    cluster GPU information
//   - maxhgo:     maximum HGO limit
//
// Returns:
//   - map[string][]string:   The nodes and gpus in the nodes.
func RetrieveUsedGPUOrderedByHGO(hasgpus map[string][]HASGPUInfo, maxhgo float64) []HASGPUInfo {
	availGPUs := make([]HASGPUInfo, 0)
	for _, nodegpus := range hasgpus {
		for _, gpu := range nodegpus {
			if gpu.HGO < float64(maxhgo) {
				availGPUs = append(availGPUs, gpu)
			}
		}
	}
	sort.Slice(availGPUs, func(i, j int) bool {
		return availGPUs[i].HGO < availGPUs[j].HGO
	})
	return availGPUs
}

// Retrieve the totally unused GPU (no pods).
// Parameters:
//   - hasgpus:    cluster GPU information
//
// Returns:
//   - map[string][]string:   The nodes and unused gpus in the nodes.
func RetrieveNewGPU(hasgpus map[string][]HASGPUInfo) map[string][]string {
	newGPUs := make(map[string][]string)
	usedGPUs := RetrieveUsedGPU(hasgpus, 100.1)
	for node, gpus := range nodeGPUMap {
		for _, gpu := range gpus {
			used := false
			for _, ugpu := range usedGPUs[node] {
				if ugpu == gpu.vGPUID {
					used = true
				}
			}
			if !used {
				newGPUs[node] = append(newGPUs[node], gpu.vGPUID)
			}
		}
	}
	return newGPUs
}

// FindOptimalRowIdx determines the best row index for a new pod on a GPU.
// It tries to minimize fragmentation by grouping pods with similar SM partition requirements.
// Parameters:
//   - gpuInfo:     The GPU information
//   - smPartition: The SM partition requirement of the new pod
//
// Returns:
//   - int:         The optimal row index for the new pod
func FindOptimalRowIdx(gpuInfo HASGPUInfo, smPartition int64) int {
	// First, try to find a row with the same SM partition configuration
	for idx, rowSM := range gpuInfo.SMConfigs {
		if rowSM == smPartition {
			// Found a row with matching SM partition
			return idx
		}
	}

	// If no matching row, look for an empty row (SM config = -1)
	for idx, rowSM := range gpuInfo.SMConfigs {
		if rowSM == -1 {
			// Found an empty row
			return idx
		}
	}

	// If all rows are used with different SM configurations,
	// find the row with the most available quota
	maxAvailQuota := int64(-1)
	bestRowIdx := 0

	for idx, pods := range gpuInfo.RowPods {
		// Calculate used quota in this row
		usedQuota := int64(0)
		for _, podName := range pods {
			usedQuota += gpuInfo.PodsConfig[podName].Quota
		}

		// Calculate available quota
		availQuota := int64(100) - usedQuota

		if availQuota > maxAvailQuota {
			maxAvailQuota = availQuota
			bestRowIdx = idx
		}
	}

	return bestRowIdx
}

// getOptimalSMPartition determines the optimal SM partition for a function based on its name and required throughput
func (r *FaSTFuncReconciler) getOptimalSMPartition(funcName string, requiredRPS float64) int64 {
	// Default fallback value
	defaultSM := int64(50)

	// Get function type from name
	funcType := r.getFunctionTypeFromName(funcName)

	// Different optimal SM partitions based on function type
	switch funcType {
	case "bert":
		if requiredRPS > 30 {
			return 70
		} else {
			return 50
		}
	case "resnet":
		if requiredRPS > 25 {
			return 50
		} else {
			return 30
		}
	case "convnext":
		// Convnext models
		if requiredRPS > 30 {
			return 70
		} else {
			return 30
		}
	case "gpt":
		// GPT models need larger SM partitions
		return 80
	case "mobilenet":
		if requiredRPS > 40 {
			return 70
		} else {
			return 30
		}
	case "rapp":
		return RappDefaultSM
	default:
		// For unknown function types, use a moderate default
		return defaultSM
	}
}

// getFunctionTypeFromName extracts the function type from the function name
func (r *FaSTFuncReconciler) getFunctionTypeFromName(funcName string) string {
	funcNameLower := strings.ToLower(funcName)

	// Check for common model types in the function name
	if strings.Contains(funcNameLower, "bert") {
		return "bert"
	} else if strings.Contains(funcNameLower, "resnet") {
		return "resnet"
	} else if strings.Contains(funcNameLower, "convnext") {
		return "convnext"
	} else if strings.Contains(funcNameLower, "gpt") {
		return "gpt"
	} else if strings.Contains(funcNameLower, "mobilenet") {
		return "mobilenet"
	} else if strings.Contains(funcNameLower, "rapp") {
		return "rapp"
	}

	// Default case
	return "unknown"
}
