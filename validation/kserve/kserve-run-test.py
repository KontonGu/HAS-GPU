#!/usr/bin/env python3
import subprocess
import time
import os
import sys
import datetime
import argparse
import re


def run_command(command, error_message=None):
    """Run a shell command and handle errors"""
    print(f"Executing: {command}")
    result = subprocess.run(command, shell=True, capture_output=True, text=True)
    if result.returncode != 0:
        print(f"Error: {error_message or 'Command failed'}")
        print(f"Error output: {result.stderr}")
        sys.exit(1)
    return result.stdout.strip()


def wait_for_pod_ready(pod_pattern, namespace="default", timeout=300):
    """Wait for a pod matching the pattern to be in Ready state"""
    print(f"Waiting for pod matching '{pod_pattern}' to be ready...")
    
    start_time = time.time()
    while time.time() - start_time < timeout:
        result = subprocess.run(
            f"kubectl get pods -n {namespace} | grep {pod_pattern}", 
            shell=True, 
            capture_output=True, 
            text=True
        )
        
        if result.returncode == 0:
            # Pod exists, check if it's ready
            pod_info = result.stdout.strip().split()
            if len(pod_info) >= 2:
                status = pod_info[1]  # The second column is typically the STATUS
                
                # Check if pod is running with all containers ready
                if "Running" in result.stdout and ("2/2" in result.stdout or "1/1" in result.stdout or "3/3" in result.stdout):
                    print(f"Pod '{pod_pattern}' is ready!")
                    return True
                else:
                    print(f"Pod status: {status}, waiting...")
        
        time.sleep(5)
    
    print(f"Timed out waiting for pod '{pod_pattern}' to be ready")
    sys.exit(1)


def update_k6_script_url(k6_script, service_name, hostname=None, args=None):
    """Update the k6 script with the correct URL and host header for the KServe service"""
    print(f"Updating k6 script with KServe URL for service '{service_name}'...")
    
    # Get the necessary information from kubectl
    ingress_host = run_command(
        "kubectl get po -l istio=ingressgateway -n istio-system -o jsonpath='{.items[0].status.hostIP}'",
        "Failed to get Istio ingress host"
    )
    
    ingress_port = run_command(
        "kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name==\"http2\")].nodePort}'",
        "Failed to get Istio ingress port"
    )
    
    # Read the k6 script
    with open(k6_script, 'r') as file:
        content = file.read()
    
    # Update the gateway URL definition - keep it simple with just the base URL
    new_gateway = f"http://{ingress_host}:{ingress_port}/predict"
    
    # Use regex to find and replace the gateway URL
    gateway_pattern = r"const\s+gateway\s*=\s*['\"]http://[^'\"]+['\"]"
    new_gateway_line = f"const gateway = '{new_gateway}'"
    
    # Check if we can find the pattern
    if re.search(gateway_pattern, content):
        updated_content = re.sub(gateway_pattern, new_gateway_line, content)
    else:
        # If we can't find the exact pattern, try a more flexible approach
        gateway_pattern = r"(const\s+gateway\s*=\s*)['\"]([^'\"]+)['\"]"
        if re.search(gateway_pattern, content):
            updated_content = re.sub(gateway_pattern, f"\\1'{new_gateway}'", content)
        else:
            print("Warning: Could not find gateway URL pattern in the k6 script.")
            print("You may need to manually update the URL in the script.")
            updated_content = content
    
    # Set hostname for KServe
    if not hostname:
        hostname = f"{service_name}.{args.namespace}.example.com"
    
    # Replace FormData usage with direct binary data approach
    formdata_pattern = r"const fd = new FormData\(\);[\s\S]*?fd\.append\([^;]*\);"
    binary_data_replacement = "// Using direct binary data instead of FormData\nconst image = open('car.jpg', 'b');"
    
    if re.search(formdata_pattern, updated_content):
        updated_content = re.sub(formdata_pattern, binary_data_replacement, updated_content)
    
    # Update the request object to use direct binary data and proper headers
    request_obj_pattern = r"let\s+resnet\s*=\s*\{[^}]*\};"
    if re.search(request_obj_pattern, updated_content):
        new_request_obj = f"""let resnet = {{
        method: 'POST',
        url: gateway,
        body: image, 
        params: {{
            headers: {{
              'Content-Type': 'application/octet-stream',
              'Accept': 'application/json',
              'Host': '{hostname}'
            }},
        }},
}};"""
        updated_content = re.sub(request_obj_pattern, new_request_obj, updated_content)
    
    # Write the updated content back to the file
    with open(k6_script, 'w') as file:
        file.write(updated_content)
    
    print(f"Updated gateway URL to: {new_gateway}")
    print(f"Updated script to use direct binary data with Host header: {hostname}")
    
    return ingress_host, ingress_port



def main():
    parser = argparse.ArgumentParser(description="Run KServe deployment and testing")
    parser.add_argument("--output-filename", required=True, help="Specify the K6 CSV output filename")
    parser.add_argument("--k6-script", default="k6-trace.js", help="Specify the K6 script file to run (default: k6-trace.js)")
    parser.add_argument("--output-dir", default=".", help="Directory to save output CSV files (default: current directory)")
    parser.add_argument("--kserve-yaml", required=True, help="Path to the KServe InferenceService YAML file")
    parser.add_argument("--pod-prefix", required=True, help="Prefix of the pod to wait for (e.g., 'resnet-server-predictor')")
    parser.add_argument("--namespace", default="default", help="Namespace for the KServe service (default: default)")
    parser.add_argument("--service-name", default="resnet-server", help="Service name for the KServe service (default: resnet-server)")
    parser.add_argument("--mode", required=True, help="Mode for the test_batch.py script (e.g., bert)")
    parser.add_argument("--requests-file", required=True, help="Path to the requests file, eg. ../client/rps.json")
    
    args = parser.parse_args()
    
    # Create a timestamp
    timestamp = datetime.datetime.now().strftime("%Y%m%d-%H%M%S")
    
    # Ensure output directory exists
    os.makedirs(args.output_dir, exist_ok=True)
    
    # 1. Deploy KServe InferenceService from YAML
    print("Step 1: Deploying KServe InferenceService...")
    run_command(f"kubectl apply -f {args.kserve_yaml}", "Failed to deploy KServe InferenceService")
    
    # 2. Wait for the predictor pod to be ready
    print("Step 2: Waiting for predictor pod to be ready...")
    wait_for_pod_ready(args.pod_prefix, args.namespace)
    
    # 3. Update k6 script with correct URL
    #print("Step 3: Updating k6 script with KServe URL...")
    #ingress_host, ingress_port = update_k6_script_url(args.k6_script, args.pod_prefix, args=args)
    time.sleep(6)

    print("Step 3: Prewarming...")
    # Get the necessary information
    ingress_host = run_command(
        "kubectl get po -l istio=ingressgateway -n istio-system -o jsonpath='{.items[0].status.hostIP}'",
        "Failed to get Istio ingress host"
    )
    
    ingress_port = run_command(
        "kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name==\"http2\")].nodePort}'",
        "Failed to get Istio ingress port"
    )
    
    service_hostname = run_command(
        f"kubectl get inferenceservice {args.service_name} -o jsonpath='{{.status.url}}' | cut -d \"/\" -f 3",
        "Failed to get service hostname"
    )
    
    # Run the test_batch.py script
    if args.mode == "bert":
        test_cmd = f"python ../client/test_batch.py --url \"http://{ingress_host}:{ingress_port}/predict\" --host \"{service_hostname}\" --mode bert --duration 5"
    elif args.mode == "image":
        test_cmd = f"python ../client/test_batch.py --url \"http://{ingress_host}:{ingress_port}/predict\" --host \"{service_hostname}\" --image \"../client/car.jpg\" --duration 5"
    run_command(test_cmd, "Failed to run test_batch.py")

    #4. running actual tests
    print("Step 4: Running test_batch.py with request counts...")
    #test_cmd = f"python ../client/test_batch.py --url \"http://{ingress_host}:{ingress_port}/predict\" --host \"{service_hostname}\" --image \"../client/car.jpg\" --mode image --request-counts-file ../trace/pressure.json --request-data-output {args.output_dir}/kserve-{args.output_filename}.csv  > ./kserve-pylog.txt"
    if args.mode == "bert":
        test_cmd = f"python ../client/test_batch.py --url \"http://{ingress_host}:{ingress_port}/predict\" --host \"{service_hostname}\" --mode bert --request-counts-file ../trace/rps.json --request-data-output {args.output_dir}/kserve-{args.output_filename}.csv  > ./kserve-pylog.txt"
    else:
        test_cmd = f"python ../client/test_batch.py --url \"http://{ingress_host}:{ingress_port}/predict\" --host \"{service_hostname}\" --image \"../client/car.jpg\" --mode image --request-counts-file ../trace/rps.json --request-data-output {args.output_dir}/kserve-{args.output_filename}.csv  > ./kserve-pylog.txt"

    run_command(test_cmd, "Failed to run test_batch.py")

    
    # 4. Run k6 test with specified output filename
    # print(f"Step 4: Running k6 test with output to {args.output_filename}...")
    # k6_output_file = os.path.join(args.output_dir, f"kserve-{args.output_filename}.csv")
    # k6_cmd = f"k6 run --out csv={k6_output_file} {args.k6_script} > ./requests/k6.txt"
    # run_command(k6_cmd, "Failed to run k6 test")
    
    # 5. Delete KServe InferenceService
    print("Step 5: Removing KServe InferenceService...")
    run_command(f"kubectl delete -f {args.kserve_yaml}", "Failed to remove KServe InferenceService")

    
    print("Step 6: Copying tmp py result...")
    run_command(f"cp ./kserve-pylog.txt {args.output_dir}/kserve-{args.output_filename}.log", "Failed to copy py log")
    
    # 6. Copy the rps script to the output directory
    #print("Step 6: Copying rps script...")
    #rps_script_copy_cmd = f"cp ../trace/rps.json {args.output_dir}/kserve-{timestamp}-{os.path.basename(args.kserve_yaml)}"
    #run_command(rps_script_copy_cmd, "Failed to copy rps script")

    # 7. Save KServe YAML to output directory
    # print("Step 7: Saving KServe YAML...")
    # yaml_copy_cmd = f"cp {args.kserve_yaml} {args.output_dir}/kserve-{timestamp}-{os.path.basename(args.kserve_yaml)}"
    # run_command(yaml_copy_cmd, "Failed to copy KServe YAML")

    print("All steps completed successfully!")
    print(f"Test results saved to: {args.output_dir}/kserve-{args.output_filename}.csv")


if __name__ == "__main__":
    main()