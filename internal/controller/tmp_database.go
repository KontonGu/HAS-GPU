package controller

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
	"net/http"
	"bytes"
)

//go:embed graph_features
var profilingFS embed.FS

// ProfilingResultEntry represents a single profiling result entry
type ProfilingResultEntry struct {
	Batch   int     `json:"batch"`
	SM      int     `json:"sm"`
	Quota   int     `json:"quota"`
	Latency float64 `json:"latency"`
}

// ProfilingData represents the entire profiling data structure
type ProfilingData struct {
	ModelName       string                 `json:"model_name"`
	ProfilingResult []ProfilingResultEntry `json:"profiling_result"`
}

// ModelData holds processed data for a specific model
type ModelData struct {
	ModelName string
	SMData    map[int]map[int]map[int]float64 // SM -> Batch -> Quota -> Latency
}

// Cache for model data to avoid repeated parsing
var (
	modelDataCache     = make(map[string]*ModelData)
	modelDataCacheLock sync.RWMutex
	modelNamesCache    []string
	initialized        bool
	initLock           sync.Mutex
)

func init() {
	// Initialize profiling data from JSON files
	if err := initializeProfilingData(); err != nil {
		// Log the error but continue - the system will use default values if profiling data is not available
		klog.Errorf("Failed to initialize profiling data: %v", err)
	}
}

// initializeProfilingData parses all embedded JSON files and populates the cache
func initializeProfilingData() error {
	initLock.Lock()
	defer initLock.Unlock()

	if initialized {
		return nil
	}

	// Get all JSON files from the embedded filesystem
	entries, err := profilingFS.ReadDir("./graph_features")
	if err != nil {
		klog.Errorf("Failed to read embedded profiling data directory, falling back to hardcoded defaults: %v", err)
		return initializeDefaultProfilingData()
	}

	// Process each JSON file
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			// Read the file content
			data, err := profilingFS.ReadFile(filepath.Join("graph_features", entry.Name()))
			if err != nil {
				return fmt.Errorf("failed to read embedded file %s: %w", entry.Name(), err)
			}

			// Parse the JSON data
			var profilingDataObj ProfilingData
			if err := json.Unmarshal(data, &profilingDataObj); err != nil {
				return fmt.Errorf("failed to parse JSON from %s: %w", entry.Name(), err)
			}

			// Get or create model data
			modelDataCacheLock.Lock()
			modelData, exists := modelDataCache[profilingDataObj.ModelName]
			if !exists {
				modelData = &ModelData{
					ModelName: profilingDataObj.ModelName,
					SMData:    make(map[int]map[int]map[int]float64),
				}
				modelDataCache[profilingDataObj.ModelName] = modelData
				modelNamesCache = append(modelNamesCache, profilingDataObj.ModelName)
			}

			// Process profiling results
			for _, result := range profilingDataObj.ProfilingResult {
				// Initialize nested maps if they don't exist
				if _, exists := modelData.SMData[result.SM]; !exists {
					modelData.SMData[result.SM] = make(map[int]map[int]float64)
				}
				if _, exists := modelData.SMData[result.SM][result.Batch]; !exists {
					modelData.SMData[result.SM][result.Batch] = make(map[int]float64)
				}

				// Store latency
				modelData.SMData[result.SM][result.Batch][result.Quota] = result.Latency
			}
			modelDataCacheLock.Unlock()
		}
	}

	initialized = true
	return nil
}

func initializeDefaultProfilingData() error {
	// Hardcoded defaults for multiple models
	defaultModels := []*ModelData{
		{
			ModelName: "resnet50",
			SMData: map[int]map[int]map[int]float64{
				30: {
					4: {
						10:  252.2491,
						20:  77.3300,
						30:  53.7186,
						40:  35.1840,
						50:  31.8869,
						70:  21.1681,
						90:  20.8474,
						100: 20.9031,
					},
				},
				50: {
					4: {
						10: 246.0224,
						20: 77.8447,
						30: 54.6511,
						40: 35.9069,
						50: 31.9668,
						70: 21.1510,
						90: 21.3813,
					},
				},
			},
		},
		{
			ModelName: "convnext-large",
			SMData: map[int]map[int]map[int]float64{
				// 30: {
				// 	4: {
				// 		10:  419.1168,
				// 		20:  142.4330,
				// 		30:  91.3830,
				// 		40:  89.3453,
				// 		50:  89.4420,
				// 		70:  89.3799,
				// 		80:  89.2816,
				// 		90:  89.3313,
				// 		100: 89.3513,
				// 	},
				// },
				50: {
					4: {
						10:  404.9708,
						20:  140.2963,
						30:  96.7166,
						40:  68.0341,
						50:  59.9109,
						70:  56.5727,
						80:  56.4936,
						90:  56.5984,
						100: 56.6035,
					},
				},
				70: {
					4: {
						30: 96.7185,
						50: 61.7110,
						70: 42.6626,
					},
				},
			},
		},
		{
			ModelName: "mobilenet-v3-large",
			SMData: map[int]map[int]map[int]float64{
				30: {
					4: {
						10:  319.7108,
						20:  92.2712,
						30:  65.4417,
						40:  45.2021,
						50:  36.2635,
						70:  26.1822,
						80:  15.0624,
						90:  13.7754,
						100: 10.7394,
					},
				},
				50: {
					4: {
						50: 26.1822,
						70: 22.6579,
						90: 13.3941,
					},
				},
			},
		},
		{
			ModelName: "shufflenet-v2-0.5",
			SMData: map[int]map[int]map[int]float64{
				50: {
					4: {
						20: 27.3490,
						50: 10.1810,
						70: 10.1810,
						90: 10.1810,
					},
				},
				70: {
					4: {
						20: 28.1956,
						50: 10.3387,
						70: 10.2795,
						90: 10.2639,
					},
				},
			},
		},
		{
			ModelName: "bert-qt",
			SMData: map[int]map[int]map[int]float64{
				30: {
					4: {
						10:  240.9081,
						20:  165.4163,
						30:  106.9170,
						40:  95.4823,
						50:  92.8406,
						60:  87.1840,
						70:  76.1591,
						80:  77.1591,
						90:  76.1591,
						100: 75.1591,
					},
				},
				50: {
					4: {
						10:  230.4163,
						20:  159.4163,
						30:  109.9170,
						40:  99.4823,
						50:  87.8406,
						70:  75.1591,
						90:  73.1591,
						100: 71.1591,
					},
				},
			},
		},
		{
			ModelName: "rapp",
			SMData: map[int]map[int]map[int]float64{
				10: {
					2: {
						20: 100.0,
						30: 100.0,
						40: 100.0,
						50: 100.0,
						60: 100.0,
						70: 100.0,
						80: 100.0,
						90: 100.0,
						100: 100.0,
					},
				},	
				20: {
					2: {
						20: 100.0,
						30: 100.0,
						40: 100.0,
						50: 100.0,
						60: 100.0,
						70: 100.0,
						80: 100.0,
						90: 100.0,
						100: 100.0,
					},
				},
				30: {
					4: {
						10:  100.0,
						20:  100.0,
						30:  100.0,
						40:  100.0,
						50:  100.0,
						60:  100.0,
						70:  100.0,
						80:  100.0,
						90:  100.0,
						100: 100.0,
					},
				},
				50: {
					4: {
						10:  100.0,
						20:  100.0,
						30:  100.0,
						40:  100.0,
						50:  100.0,
						60:  100.0,
						70:  100.0,
						80:  100.0,
						90:  100.0,
						100: 100.0,
					},
				},
			},
		},
	}

	// Add all default models to the cache
	for _, model := range defaultModels {
		modelDataCache[model.ModelName] = model
		modelNamesCache = append(modelNamesCache, model.ModelName)
	}

	initialized = true
	return nil
}

// extractModelName extracts the model name from a function name
func extractModelName(funcname string) string {
	klog.Info("extractModelName called with", "funcname", funcname, "type", fmt.Sprintf("%T", funcname))
	
	if funcname == "" {
		klog.Info("Empty function name provided, defaulting to resnet50")
		return "resnet50" // Default model
	}

	// Special case for rapp - check this first before any other processing
	lowerName := strings.ToLower(funcname)
	klog.Info("Checking for rapp in function name", "lowerName", lowerName, "contains_rapp", strings.Contains(lowerName, "rapp"))
	
	if lowerName == "rapp" || strings.Contains(lowerName, "rapp") {
		klog.Info("Detected rapp model", "function", funcname)
		return "rapp"
	}

	// Initialize profiling data if needed
	if err := initializeProfilingData(); err != nil {
		klog.Errorf("Failed to initialize profiling data in extractModelName: %v", err)
		return "resnet50" // Default to resnet50 on error
	}

	// Get available model names
	modelDataCacheLock.RLock()
	defer modelDataCacheLock.RUnlock()

	// Log available models for debugging
	klog.Info("Available models in cache", "models", modelNamesCache)

	// First try exact matches with model names
	for _, modelName := range modelNamesCache {
		if strings.Contains(lowerName, strings.ToLower(modelName)) {
			klog.Info("Found exact model match", "model", modelName)
			return modelName
		}
	}

	// If no exact match, try some common mappings
	modelMappings := map[string]string{
		"resnet":     "resnet50",
		"convnext":   "convnext-large",
		"mobilenet":  "mobilenet_v3_large",
		"shufflenet": "shufflenet_v2_x0_5",
		"bert":       "bert-qt",
		// Add more mappings as needed
	}

	for prefix, modelName := range modelMappings {
		if strings.Contains(lowerName, prefix) {
			klog.Info("Found model mapping match", "prefix", prefix, "modelName", modelName)
			return modelName
		}
	}

	// Check if the function name contains a version number after a model type
	// For example: "resnet101" -> "resnet50" (as we don't have resnet101 profiling data)
	for prefix, modelName := range modelMappings {
		if strings.Contains(lowerName, prefix) && containsNumber(lowerName[strings.Index(lowerName, prefix):]) {
			return modelName
		}
	}

	// Default to rapp if no match
	klog.Info("No model match found for function name, defaulting to rapp", "function", funcname)
	return "rapp"
}

// containsNumber checks if a string contains any digits
func containsNumber(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

// containsString checks if a string slice contains a specific string
func containsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

func rapp(funcname string, batch int64, smPartition int64, quota int64) float64 {
	// Add debug log at function entry
	klog.Warningf("[Testing RAPP] function called", 
		"function", funcname, 
		"batch", batch, 
		"sm", smPartition, 
		"quota", quota)

	// Force model name to "rapp" if the function name contains "rapp"
	var modelName string
	if strings.Contains(strings.ToLower(funcname), "rapp") {
		modelName = "rapp"
		klog.Info("Forced model name to rapp", "function", funcname)
	} else {
		// Extract model name from function name
		modelName = extractModelName(funcname)
		klog.Info("Model name extracted", "function", funcname, "model", modelName)
	}

	// Convert parameters to integers for lookup
	quotaInt := int(quota)
	smInt := int(smPartition)
	batchInt := int(batch)

	// If batch is 0, use default batch size for the model
	if batchInt == 0 {
		// Default batch sizes for different models
		defaultBatchSizes := map[string]int{
			"resnet50":           4,
			"convnext-large":     4,
			"mobilenet_v3_large": 4,
			"bert-qt":            4,
			"shufflenet_v2_x0_5": 8,
			"rapp":               2, // Add default batch size for rapp
			// Add more defaults as needed
		}

		if defaultBatch, exists := defaultBatchSizes[modelName]; exists {
			batchInt = defaultBatch
			klog.Info("Using default batch size for model", "model", modelName, "batch", batchInt)
		} else {
			batchInt = 4 // General default
			klog.Info("Using general default batch size", "model", modelName, "batch", batchInt)
		}
	}

	// Special handling for rapp model - always use the predefined configuration
	if modelName == "rapp" || strings.Contains(strings.ToLower(funcname), "rapp") {
		klog.Info("Using predefined configuration for rapp model",
			"model", modelName,
			"sm", smInt,
			"batch", batchInt,
			"quota", quotaInt)
		
		// Use default SM partition if not specified
		if smInt == 0 {
			smInt = int(RappDefaultSM)
			klog.Info("Using default SM partition for rapp model", "sm", smInt)
		}
		
		// Use default batch size if not specified
		if batchInt == 0 {
			batchInt = int(RappDefaultBatchSize)
			klog.Info("Using default batch size for rapp model", "batch", batchInt)
		}
		
		// Use default quota if not specified
		if quotaInt == 0 {
			quotaInt = int(RappDefaultQuota)
			klog.Info("Using default quota for rapp model", "quota", quotaInt)
		}
		
		// Predefined throughput value for rapp model
		predefinedThroughput := 100.0 
		
		// Predefined latency value for rapp model
		predefinedLatency := 10.0 
		
		// Calculate adjusted throughput using the same formula as in the remote call path
		adjustedThroughput := (1000.0 / (predefinedLatency + 70)) * float64(batchInt+1) / 2
		
		klog.Info("Using predefined rapp configuration",
			"model", modelName,
			"sm", smInt,
			"batch", batchInt,
			"quota", quotaInt,
			"predefined_latency", predefinedLatency,
			"adjusted_latency", predefinedLatency + 70,
			"predefined_throughput", predefinedThroughput,
			"adjusted_throughput", adjustedThroughput)
		
		return adjustedThroughput
	}
	
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Create the payload for the RAPP service
	payload := map[string]interface{}{
		"model_name": modelName,
		"batch_size": batchInt,
		"sm":         smInt,
		"quota":      quotaInt,
	}

	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		klog.Errorf("Failed to marshal JSON payload for RAPP service: %v", err)
		// Fallback to the old method if HTTP request fails
		return getThroughput(modelName, smInt, batchInt, quotaInt)
	}

	// URL for the RAPP service - use Kubernetes service URL
	url := "http://rapp.fast-gshare-fn.svc.cluster.local:8080/rapp_query"
	
	// If the model is "bert-qt", use the old implementation instead of calling the remote RAPP service
	if modelName == "bert-qt" {
		klog.Info("Using local calculation for bert model",
			"model", modelName,
			"sm", smInt,
			"batch", batchInt,
			"quota", quotaInt)
		
		localThroughput := getThroughput(modelName, smInt, batchInt, quotaInt)
		
		klog.Info("Used local calculation for bert model",
			"model", modelName,
			"sm", smInt,
			"batch", batchInt,
			"quota", quotaInt,
			"throughput", localThroughput)
		
		return localThroughput
	}
	
	klog.Info("Sending request to RAPP service", 
		"url", url, 
		"payload", string(jsonPayload))

	// Make the HTTP request
	startTime := time.Now()
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	requestDuration := time.Since(startTime)
	
	if err != nil {
		klog.Info("RAPP service unavailable, falling back to local calculation", 
			"url", url, 
			"model", modelName, 
			"sm", smInt, 
			"batch", batchInt, 
			"quota", quotaInt,
			"error", err.Error(),
			"request_duration_ms", requestDuration.Milliseconds())
		// Fallback to the old method if HTTP request fails
		return getThroughput(modelName, smInt, batchInt, quotaInt)
	}
	defer resp.Body.Close()

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		klog.Info("RAPP service returned non-OK status, falling back to local calculation", 
			"statusCode", resp.StatusCode, 
			"url", url, 
			"model", modelName, 
			"sm", smInt, 
			"batch", batchInt, 
			"quota", quotaInt,
			"response_body", bodyString,
			"request_duration_ms", requestDuration.Milliseconds())
		// Fallback to the old method if HTTP request fails
		return getThroughput(modelName, smInt, batchInt, quotaInt)
	}

	// Read and parse the response
	var responseData map[string]interface{}
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyString := string(bodyBytes)
	
	if err := json.Unmarshal(bodyBytes, &responseData); err != nil {
		klog.Info("Failed to decode RAPP service response, falling back to local calculation", 
			"url", url, 
			"model", modelName, 
			"sm", smInt, 
			"batch", batchInt, 
			"quota", quotaInt,
			"error", err.Error(),
			"response_body", bodyString,
			"request_duration_ms", requestDuration.Milliseconds())
		// Fallback to the old method if response parsing fails
		return getThroughput(modelName, smInt, batchInt, quotaInt)
	}

	klog.Info("Received response from RAPP service",
		"url", url,
		"model", modelName,
		"sm", smInt,
		"batch", batchInt,
		"quota", quotaInt,
		"response", responseData,
		"request_duration_ms", requestDuration.Milliseconds())

	// Check if the response has the expected structure
	status, statusOk := responseData["status"].(string)
	if !statusOk || status != "success" {
		klog.Info("Invalid status in RAPP service response, falling back to local calculation", 
			"response", responseData, 
			"model", modelName, 
			"sm", smInt, 
			"batch", batchInt, 
			"quota", quotaInt)
		return getThroughput(modelName, smInt, batchInt, quotaInt)
	}

	// Extract the predict section from the response
	predictData, predictOk := responseData["predict"].(map[string]interface{})
	if !predictOk {
		klog.Info("Missing predict data in RAPP service response, falling back to local calculation", 
			"response", responseData, 
			"model", modelName, 
			"sm", smInt, 
			"batch", batchInt, 
			"quota", quotaInt)
		return getThroughput(modelName, smInt, batchInt, quotaInt)
	}

	// Extract latency from the predict section
	latency, latencyOk := predictData["latency"].(float64)
	if !latencyOk || latency <= 0 {
		klog.Info("Invalid latency in RAPP service response, falling back to local calculation", 
			"predict_data", predictData, 
			"model", modelName, 
			"sm", smInt, 
			"batch", batchInt, 
			"quota", quotaInt)
		return getThroughput(modelName, smInt, batchInt, quotaInt)
	}
	
	// Calculate throughput from the latency
	adjustedThroughput := (1000.0 / (latency + 70)) * float64(batchInt+1) / 2
	
	klog.Info("Calculated throughput with preprocessing time",
		"model", modelName,
		"sm", smInt,
		"batch", batchInt,
		"quota", quotaInt,
		"raw_latency", latency,
		"adjusted_latency", latency + 70,
		"adjusted_throughput", adjustedThroughput)
	
	return adjustedThroughput
}

// GetMostEfficientConfig returns the most efficient configuration for a given function
// If no parameters are provided, it uses default values (resnet50 with 50 RPS)
func GetMostEfficientConfig(funcname ...string) (int64, int64, int) {
	var modelName string
	
	// Check if a function name is provided
	if len(funcname) > 0 && funcname[0] != "" {
		// Check for rapp directly before calling extractModelName
		if strings.Contains(strings.ToLower(funcname[0]), "rapp") {
			modelName = "rapp"
			klog.Info("Detected rapp model directly in GetMostEfficientConfig", "function", funcname[0])
			
			// For rapp models, return the default configuration directly
			return RappDefaultSM, RappDefaultQuota, int(RappDefaultBatchSize)
		} else {
			// Extract model name from function name
			modelName = extractModelName(funcname[0])
			klog.Info("Finding most efficient config", "function", funcname, "model", modelName)
		}
	} else {
		// If no function name provided, use default model
		modelName = "resnet50"
		klog.Info("No function name provided, using default model", "model", modelName)
	}
	
	// Default throughput requirement
	requiredThroughput := 35.0

	// Default configuration if nothing else works
	bestSM := 30
	bestBatch := 4
	bestQuota := 30
	bestEfficiency := 0.0

	// Initialize profiling data
	if err := initializeProfilingData(); err != nil {
		klog.Errorf("Error initializing profiling data in GetMostEfficientConfig: %v", err)
		return int64(bestSM), int64(bestQuota), bestBatch
	}

	modelDataCacheLock.RLock()
	defer modelDataCacheLock.RUnlock()

	// Get model data
	modelData, exists := modelDataCache[modelName]
	if !exists {
		klog.Info("No profiling data found for model, using defaults", "model", modelName)
		return int64(bestSM), int64(bestQuota), bestBatch
	}

	// Try different configurations to find the most efficient one
	for sm, batchMap := range modelData.SMData {
		for batch, quotaMap := range batchMap {
			for quota := range quotaMap {
				// Use rapp function to get throughput instead of getThroughput
				// This will provide more accurate predictions by using the same function
				// that's used during scaling operations
				throughput := rapp(funcname[0], int64(batch), int64(sm), int64(quota))

				// Skip configurations that don't meet the throughput requirement
				if throughput < requiredThroughput {
					continue
				}

				// Calculate efficiency (throughput per resource unit)
				// Resource units = SM * Quota
				resourceUnits := float64(sm * quota)
				efficiency := throughput / resourceUnits

				// Update best configuration if this one is more efficient
				if efficiency > bestEfficiency {
					bestEfficiency = efficiency
					bestSM = sm
					bestBatch = batch
					bestQuota = quota
				}
			}
		}
	}

	klog.Info("Found most efficient config using rapp function",
		"model", modelName,
		"sm", bestSM,
		"batch", bestBatch,
		"quota", bestQuota,
		"efficiency", bestEfficiency)

	return int64(bestSM), int64(bestQuota), bestBatch
}

// getModelNames returns the names of all available models
func getModelNames() []string {
	if err := initializeProfilingData(); err != nil {
		fmt.Printf("Error initializing profiling data: %v\n", err)
		return []string{}
	}

	modelDataCacheLock.RLock()
	defer modelDataCacheLock.RUnlock()

	// Return a copy to prevent modification
	result := make([]string, len(modelNamesCache))
	copy(result, modelNamesCache)
	return result
}

// getLatency returns the latency for the given model, SM partition, batch size, and quota
func getLatency(model string, sm, batch, quota int) float64 {
	if err := initializeProfilingData(); err != nil {
		fmt.Printf("Error initializing profiling data: %v\n", err)
		return -1
	}
	modelDataCacheLock.RLock()
	defer modelDataCacheLock.RUnlock()

	// Get model data
	modelData, exists := modelDataCache[model]
	if !exists {
		return -1
	}

	// Get latency
	if smMap, ok := modelData.SMData[sm]; ok {
		if batchMap, ok := smMap[batch]; ok {
			if latency, ok := batchMap[quota]; ok {
				return latency
			}
		}
	}
	return -1
}

// getThroughput returns the throughput (requests per second) for the given model, SM partition, batch size, and quota
func getThroughput(model string, sm, batch, quota int) float64 {
	latency := getLatency(model, sm, batch, quota)
	if latency <= 0 {
		return 0
	}
	// add 30ms to latency to account for preprocessing and http request sending time
	return (1000.0 / (latency + 70)) * float64(batch+1) / 2 // Convert ms to seconds and multiply by batch size
}
