import requests
import time
import threading
from concurrent.futures import ThreadPoolExecutor
import os
import argparse
from datetime import datetime
import json
import numpy as np
import statistics
import csv
import random

# Global list to store request times
request_times = []
request_times_lock = threading.Lock()

# Global list to store detailed request data
request_data = []
request_data_lock = threading.Lock()

def send_request(image_path="car.jpg", url=None, host_header=None, mode="image", max_new_tokens=128):
    if url is None:
        url = "http://localhost:5001/predict"
    
    files = None
    data = None
    headers = {"Accept": "application/json"}
    
    # Add host header for KServe if provided
    if host_header:
        headers["Host"] = host_header
    
    if mode == "bert":
        # BERT question-answering request
        sample = {
            "question": "What is the capital of France?",
            "context": "Paris is the capital and largest city of France. It is situated on the river Seine."
        }
        
        headers["Content-Type"] = "application/json"
        try:
            start_time = time.time()
            response = requests.post(url, json=sample, headers=headers, timeout=10)
            elapsed = time.time() - start_time
            
            # Record request time
            with request_times_lock:
                request_times.append(elapsed)
            
            # Record detailed request data
            with request_data_lock:
                request_data.append({
                    "timestamp": datetime.now().isoformat(),
                    "duration": elapsed,
                    "status_code": response.status_code,
                    "mode": mode,
                    "success": response.status_code == 200
                })
                
            print(f"BERT Request sent at {time.strftime('%H:%M:%S')} - Status: {response.status_code}")
            print(f"Question: {sample['question']}")
            print(f"Time taken: {elapsed:.3f} seconds")
            return response.json() if response.status_code == 200 else response.text
        except Exception as e:
            # Record failed request
            with request_data_lock:
                request_data.append({
                    "timestamp": datetime.now().isoformat(),
                    "duration": time.time() - start_time if 'start_time' in locals() else 0,
                    "status_code": 0,  # 0 indicates connection error
                    "mode": mode,
                    "success": False,
                    "error": str(e)
                })
            print(f"Error sending BERT request: {e}")
            return None
    elif mode == "gptj":
        # GPT-J text generation request
        payload = {
            "texts": ["Write a short story about a robot learning to paint:"],
            "max_new_tokens": max_new_tokens  # Add default max_new_tokens parameter
        }
        
        headers["Content-Type"] = "application/json"
        try:
            start_time = time.time()
            response = requests.post(url, json=payload, headers=headers)
            elapsed = time.time() - start_time
            
            # Record request time
            with request_times_lock:
                request_times.append(elapsed)
            
            # Record detailed request data
            with request_data_lock:
                request_data.append({
                    "timestamp": datetime.now().isoformat(),
                    "duration": elapsed,
                    "status_code": response.status_code,
                    "mode": mode,
                    "success": response.status_code == 200
                })
            
            print(f"GPT-J Request sent at {time.strftime('%H:%M:%S')} - Status: {response.status_code}")
            print(f"Time taken: {elapsed:.3f} seconds")
            
            if response.status_code == 200:
                result = response.json()
                # Handle response format consistently with GPTJClient
                if "generated_texts" in result:
                    result["outputs"] = result.pop("generated_texts")
                result["timing"] = {"total_time": elapsed}
                return result
            return response.text
        except Exception as e:
            # Record failed request
            with request_data_lock:
                request_data.append({
                    "timestamp": datetime.now().isoformat(),
                    "duration": time.time() - start_time if 'start_time' in locals() else 0,
                    "status_code": 0,  # 0 indicates connection error
                    "mode": mode,
                    "success": False,
                    "error": str(e)
                })
            print(f"Error sending GPT-J request: {e}")
            return None
    else:
        # Image-based request (original functionality)
        # Determine if we're using KServe or local endpoint
        if "kserve" in url or "v1/models" in url:
            # KServe expects a different format
            with open(image_path, "rb") as img_file:
                img_bytes = img_file.read()
                
            # For KServe, we need to send binary data
            headers["Content-Type"] = "application/octet-stream"
            data = img_bytes
        else:
            # Original format for local endpoint
            files = {"payload": open(image_path, "rb")}
        
        try:
            start_time = time.time()
            if files:
                response = requests.post(url, files=files, headers=headers)
            else:
                response = requests.post(url, data=data, headers=headers)
            elapsed = time.time() - start_time
            
            # Record request time
            with request_times_lock:
                request_times.append(elapsed)
            
            # Record detailed request data
            with request_data_lock:
                request_data.append({
                    "timestamp": datetime.now().isoformat(),
                    "duration": elapsed,
                    "status_code": response.status_code,
                    "mode": mode,
                    "success": response.status_code == 200
                })
                
            print(f"Image Request sent at {time.strftime('%H:%M:%S')} - Status: {response.status_code}")
            return response.json() if response.status_code == 200 else response.text
        except Exception as e:
            # Record failed request
            with request_data_lock:
                request_data.append({
                    "timestamp": datetime.now().isoformat(),
                    "duration": time.time() - start_time if 'start_time' in locals() else 0,
                    "status_code": 0,  # 0 indicates connection error
                    "mode": mode,
                    "success": False,
                    "error": str(e)
                })
            print(f"Error sending image request: {e}")
            return None

def send_bert_request(url=None, host_header=None, sample_idx=None):
    """
    Send a BERT question-answering request
    
    Args:
        url: Target URL
        host_header: Optional host header for KServe
        sample_idx: Index of the sample question to use (0-2). If None, a random sample will be selected.
    """
    if url is None:
        url = "http://localhost:5001/predict"
    
    # Sample questions and contexts
    samples = [
        {
            "question": "What is the capital of France?",
            "context": "Paris is the capital and largest city of France. It is situated on the river Seine."
        },
        {
            "question": "Who wrote Romeo and Juliet?",
            "context": "Romeo and Juliet is a tragedy written by William Shakespeare early in his career about two young Italian lovers."
        },
        {
            "question": "What is the speed of light?",
            "context": "The speed of light in vacuum, commonly denoted as c, is a universal physical constant that is exactly equal to 299,792,458 metres per second."
        }
    ]
    
    # Randomly select a sample if sample_idx is None
    if sample_idx is None:
        sample_idx = random.randint(0, len(samples) - 1)
    
    # Use modulo to ensure valid index even if out of range
    sample = samples[sample_idx % len(samples)]
    
    headers = {"Accept": "application/json", "Content-Type": "application/json"}
    if host_header:
        headers["Host"] = host_header
    
    try:
        start_time = time.time()
        response = requests.post(url, json=sample, headers=headers)
        elapsed = time.time() - start_time
        
        # Record request time
        with request_times_lock:
            request_times.append(elapsed)
        
        # Record detailed request data
        with request_data_lock:
            request_data.append({
                "timestamp": datetime.now().isoformat(),
                "duration": elapsed,
                "status_code": response.status_code,
                "mode": "bert",
                "success": response.status_code == 200
            })
        
        print(f"BERT Request sent at {time.strftime('%H:%M:%S')} - Status: {response.status_code}")
        print(f"Question: {sample['question']}")
        print(f"Time taken: {elapsed:.3f} seconds")
        
        return response.json() if response.status_code == 200 else response.text
    except Exception as e:
        # Record failed request
        with request_data_lock:
            request_data.append({
                "timestamp": datetime.now().isoformat(),
                "duration": time.time() - start_time if 'start_time' in locals() else 0,
                "status_code": 0,  # 0 indicates connection error
                "mode": "bert",
                "success": False,
                "error": str(e)
            })
        print(f"Error sending BERT request: {e}")
        return None

def send_gptj_request(url=None, host_header=None, prompt_idx=0, max_new_tokens=128):
    """
    Send a GPT-J text generation request
    
    Args:
        url: Target URL
        host_header: Optional host header for KServe
        prompt_idx: Index of the prompt to use (0-2)
        max_new_tokens: Maximum number of new tokens to generate
    """
    if url is None:
        url = "http://localhost:5001/predict"
    
    # Sample prompts for GPT-J
    prompts = [
        "Write a short story about a robot learning to paint:",
        "What is artificial intelligence?",
        "Write a haiku about spring:"
    ]
    
    # Use modulo to ensure valid index even if out of range
    prompt = prompts[prompt_idx % len(prompts)]
    
    payload = {
        "texts": [prompt],
        "max_new_tokens": max_new_tokens
    }
    
    headers = {"Accept": "application/json", "Content-Type": "application/json"}
    if host_header:
        headers["Host"] = host_header
    
    try:
        start_time = time.time()
        response = requests.post(url, json=payload, headers=headers)
        elapsed = time.time() - start_time
        
        # Record request time
        with request_times_lock:
            request_times.append(elapsed)
        
        # Record detailed request data
        with request_data_lock:
            request_data.append({
                "timestamp": datetime.now().isoformat(),
                "duration": elapsed,
                "status_code": response.status_code,
                "mode": "gptj",
                "success": response.status_code == 200
            })
        
        print(f"GPT-J Request sent at {time.strftime('%H:%M:%S')} - Status: {response.status_code}")
        print(f"Prompt: {prompt}")
        print(f"Time taken: {elapsed:.3f} seconds")
        
        if response.status_code == 200:
            result = response.json()
            # Handle response format consistently with GPTJClient
            if "generated_texts" in result:
                result["outputs"] = result.pop("generated_texts")
                print(f"Generated text: {result['outputs'][0][:100]}...")  # Print first 100 chars
            elif "outputs" in result:
                print(f"Generated text: {result['outputs'][0][:100]}...")  # Print first 100 chars
            
            # Add timing information
            result["timing"] = {"total_time": elapsed}
            return result
        else:
            return response.text
    except Exception as e:
        # Record failed request
        with request_data_lock:
            request_data.append({
                "timestamp": datetime.now().isoformat(),
                "duration": time.time() - start_time if 'start_time' in locals() else 0,
                "status_code": 0,  # 0 indicates connection error
                "mode": "gptj",
                "success": False,
                "error": str(e)
            })
        print(f"Error sending GPT-J request: {e}")
        return None

def send_multiple_requests(num_requests=10, delay=0.1, url=None, host_header=None, mode="image", image_path="car.jpg", max_new_tokens=128):
    print(f"Sending {num_requests} requests with {delay}s delay between them")
    
    for i in range(num_requests):
        if mode == "bert":
            # Rotate through the sample questions
            threading.Thread(target=send_bert_request, kwargs={
                "url": url, 
                "host_header": host_header, 
                "sample_idx": i % 3
            }).start()
        elif mode == "gptj":
            # Rotate through the sample prompts
            threading.Thread(target=send_gptj_request, kwargs={
                "url": url, 
                "host_header": host_header, 
                "prompt_idx": i % 3,
                "max_new_tokens": max_new_tokens
            }).start()
        else:
            threading.Thread(target=send_request, kwargs={
                "url": url, 
                "host_header": host_header, 
                "image_path": image_path,
                "mode": mode,
                "max_new_tokens": max_new_tokens
            }).start()
        time.sleep(delay)  # Add small delay between requests

def send_concurrent_requests(num_requests=10, url=None, host_header=None, mode="image", image_path="car.jpg", max_new_tokens=128):
    """
    Send multiple requests concurrently using a thread pool
    """
    print(f"Sending {num_requests} concurrent requests")
    
    with ThreadPoolExecutor(max_workers=num_requests) as executor:
        futures = []
        
        for i in range(num_requests):
            if mode == "bert":
                futures.append(executor.submit(
                    send_bert_request, 
                    url=url, 
                    host_header=host_header, 
                    sample_idx=i % 3
                ))
            elif mode == "gptj":
                futures.append(executor.submit(
                    send_gptj_request, 
                    url=url, 
                    host_header=host_header, 
                    prompt_idx=i % 3,
                    max_new_tokens=max_new_tokens
                ))
            else:
                futures.append(executor.submit(
                    send_request, 
                    url=url, 
                    host_header=host_header, 
                    image_path=image_path,
                    mode=mode,
                    max_new_tokens=max_new_tokens
                ))

def send_rate_limited_requests(requests_per_second=10, duration_seconds=60, url=None, host_header=None, 
                              mode="image", image_path="car.jpg", max_new_tokens=128, wait_for_completion=True):
    """
    Send requests at a fixed rate using multiple threads.
    
    Args:
        requests_per_second: Number of requests to send per second
        duration_seconds: How long to run the test for
        url: Target URL
        host_header: Optional host header for KServe
        mode: Type of request ("image", "bert", or "gptj")
        image_path: Path to image file (only used for image mode)
        max_new_tokens: Maximum number of new tokens to generate (for GPT-J only)
        wait_for_completion: Whether to wait for requests to complete at the end
    """
    print(f"Sending {requests_per_second} requests per second for {duration_seconds} seconds")
    
    # Calculate total requests
    total_requests = requests_per_second * duration_seconds

    # Calculate delay between requests to distribute them evenly within each second
    delay_between_requests = 1.0 / requests_per_second if requests_per_second > 0 else 0
    
    # Maximum number of retries for failed requests
    max_retries = 3
    
    # Create a thread pool with enough workers
    with ThreadPoolExecutor(max_workers=requests_per_second * 2) as executor:
        start_time = time.time()
        request_count = 0
        success_count = 0
        failure_count = 0
        all_futures = []  # Track all futures
        
        while time.time() - start_time < duration_seconds:
            batch_start = time.time()
            
            # Submit batch of requests for this second
            for i in range(requests_per_second):
                def send_with_retry(retry_count=0):
                    try:
                        if mode == "bert":
                            future = executor.submit(
                                send_bert_request, 
                                url=url, 
                                host_header=host_header, 
                                sample_idx=request_count % 3
                            )
                            all_futures.append(future)
                        elif mode == "gptj":
                            future = executor.submit(
                                send_gptj_request, 
                                url=url, 
                                host_header=host_header, 
                                prompt_idx=request_count % 3,
                                max_new_tokens=max_new_tokens
                            )
                            all_futures.append(future)
                        else:
                            future = executor.submit(
                                send_request, 
                                url=url, 
                                host_header=host_header, 
                                image_path=image_path,
                                mode=mode,
                                max_new_tokens=max_new_tokens
                            )
                            all_futures.append(future)
                        nonlocal success_count
                        success_count +=1
                        return True
                    except Exception as e:
                        if retry_count < max_retries:
                            # Exponential backoff: wait longer for each retry
                            backoff_time = 0.1 * (2 ** retry_count)
                            print(f"Request failed: {e}. Retrying in {backoff_time:.2f}s (attempt {retry_count+1}/{max_retries})")
                            time.sleep(backoff_time)
                            return send_with_retry(retry_count + 1)
                        else:
                            nonlocal failure_count
                            failure_count += 1
                            print(f"Request failed after {max_retries} retries: {e}")
                            return None
                
                # Submit the send_with_retry function but don't wait for it to complete
                future = executor.submit(send_with_retry)
                all_futures.append(future)
                request_count += 1

                # Add a small delay between requests to spread them out evenly
                time.sleep(delay_between_requests)
            
            # Wait for the remaining time in this second
            elapsed = time.time() - batch_start
            if elapsed < 1.0:
                time.sleep(1.0 - elapsed)
        
        print(f"Duration of {duration_seconds} seconds reached. Sent {request_count} requests.")
        
        # Only wait for requests to complete if wait_for_completion is True
        if wait_for_completion:
            print(f"Waiting for {len(all_futures)} in-flight requests to complete...")
            
            # Wait for all futures to complete
            for future in all_futures:
                try:
                    future.result()
                except Exception as e:
                    # Just log exceptions, don't stop waiting for other futures
                    print(f"Exception in request: {e}")
            
            print(f"All requests completed. Total: {request_count} requests over {duration_seconds} seconds")
        else:
            print(f"Not waiting for requests to complete. Total: {request_count} requests over {duration_seconds} seconds")
        
        return all_futures

def run_request_counts(request_counts_file, duration_per_stage=2.0, url=None, host_header=None, 
                      mode="image", image_path="car.jpg", max_new_tokens=128):
    """
    Run load test based on request counts read from a JSON file
    
    Args:
        request_counts_file: Path to JSON file containing processed_request_counts
        duration_per_stage: Duration in seconds to maintain each request count level
        url: Target URL
        host_header: Optional host header for KServe
        mode: Type of request ("image", "bert", or "gptj")
        image_path: Path to image file (only used for image mode)
        max_new_tokens: Maximum number of new tokens to generate (for GPT-J only)
    """
    try:
        # Load JSON data
        with open(request_counts_file, 'r') as f:
            data = json.load(f)
            
        # Extract processed_request_counts
        if 'processed_request_counts' in data:
            request_counts = data['processed_request_counts']
        else:
            # If the JSON file directly contains an array
            request_counts = data
            
        if not request_counts:
            print("No request counts found in the JSON file")
            return
            
        print(f"\nStarting request counts-based load test with {len(request_counts)} stages...")
        print(f"Each stage will run for {duration_per_stage} seconds")
        print(f"URL: {url}")
        if host_header:
            print(f"Host header: {host_header}")
        print(f"Mode: {mode}")
        
        # Start time for the test
        start_time = time.time()
        all_futures = []
        
        # Process each request count
        for i, count in enumerate(request_counts):
            count = int(count)  # Ensure count is an integer
            
            # Log the stage
            elapsed = time.time() - start_time
            print(f"[{elapsed:.3f}s] Stage {i+1}/{len(request_counts)}: Setting request rate to {count} requests/second")
            
            # Run rate-limited requests for this stage
            # Only wait for completion on the very last stage
            is_last_stage = (i == len(request_counts) - 1)
            futures = send_rate_limited_requests(
                requests_per_second=count, 
                duration_seconds=duration_per_stage,
                url=url, 
                host_header=host_header, 
                mode=mode, 
                image_path=image_path,
                max_new_tokens=max_new_tokens,
                wait_for_completion=is_last_stage
            )
            
            # If not the last stage, collect futures to potentially handle later
            if not is_last_stage:
                all_futures.extend(futures)
            
        # Return total elapsed time
        return time.time() - start_time
            
    except FileNotFoundError:
        print(f"Error: JSON file '{request_counts_file}' not found.")
    except json.JSONDecodeError:
        print(f"Error: '{request_counts_file}' is not a valid JSON file.")
    except Exception as e:
        print(f"Error processing request counts: {str(e)}")

def calculate_latency_stats():
    """
    Calculate latency statistics from recorded request times
    
    Returns:
        dict: Dictionary containing avg, p90, p95, p99 latency values
    """
    if not request_times:
        return {
            "avg_latency": 0,
            "p90_latency": 0,
            "p95_latency": 0,
            "p99_latency": 0,
            "min_latency": 0,
            "max_latency": 0,
            "total_requests": 0
        }
    
    # Calculate statistics
    avg_latency = sum(request_times) / len(request_times)
    sorted_times = sorted(request_times)
    p90_index = int(len(sorted_times) * 0.90)
    p95_index = int(len(sorted_times) * 0.95)
    p99_index = int(len(sorted_times) * 0.99)
    
    return {
        "avg_latency": avg_latency,
        "p90_latency": sorted_times[p90_index] if p90_index < len(sorted_times) else sorted_times[-1],
        "p95_latency": sorted_times[p95_index] if p95_index < len(sorted_times) else sorted_times[-1],
        "p99_latency": sorted_times[p99_index] if p99_index < len(sorted_times) else sorted_times[-1],
        "min_latency": sorted_times[0],
        "max_latency": sorted_times[-1],
        "total_requests": len(request_times)
    }

def save_latency_results(stats, filename="latency_results.json"):
    """
    Save latency statistics to a JSON file
    
    Args:
        stats: Dictionary of latency statistics
        filename: Output filename
    """
    with open(filename, 'w') as f:
        json.dump(stats, f, indent=2)
    print(f"Latency results saved to {filename}")

def save_request_data(filename="request_data.json", format="json"):
    """
    Save detailed request data to a file in JSON or CSV format
    
    Args:
        filename: Output filename
        format: Output format ('json' or 'csv')
    """
    if not request_data:
        print("No request data to save")
        return
    
    try:
        if format.lower() == "json":
            with open(filename, 'w') as f:
                json.dump({
                    "timestamp": datetime.now().isoformat(),
                    "total_requests": len(request_data),
                    "requests": request_data
                }, f, indent=2)
            print(f"Request data saved to {filename}")
        elif format.lower() == "csv":
            csv_filename = filename.replace('.json', '.csv') if filename.endswith('.json') else filename
            with open(csv_filename, 'w', newline='') as f:
                # Determine all possible fields from the data
                fieldnames = set()
                for req in request_data:
                    fieldnames.update(req.keys())
                
                writer = csv.DictWriter(f, fieldnames=sorted(list(fieldnames)))
                writer.writeheader()
                for req in request_data:
                    writer.writerow(req)
            print(f"Request data saved to {csv_filename}")
        else:
            print(f"Unsupported format: {format}. Use 'json' or 'csv'.")
    except Exception as e:
        print(f"Error saving request data: {e}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Send batch requests to inference service')
    parser.add_argument('--url', type=str, default="http://localhost:5001/predict", 
                        help='URL for the inference service')
    parser.add_argument('--host', type=str, default=None, 
                        help='Host header for KServe (e.g., resnet-server.default.example.com)')
    parser.add_argument('--image', type=str, default="car.jpg", 
                        help='Path to the image file (for image mode)')
    parser.add_argument('--sequential', type=int, default=10, 
                        help='Number of sequential requests to send')
    parser.add_argument('--concurrent', type=int, default=10, 
                        help='Number of concurrent requests to send')
    parser.add_argument('--delay', type=float, default=0.1, 
                        help='Delay between sequential requests in seconds')
    parser.add_argument('--rate', type=int, default=5,
                        help='Number of requests per second for rate-limited testing')
    parser.add_argument('--duration', type=int, default=120,
                        help='Duration in seconds for rate-limited testing')
    parser.add_argument('--request-counts-file', 
                        help='JSON file containing processed_request_counts')
    parser.add_argument('--request-counts-duration', type=float, default=5.0,
                        help='Duration in seconds for each request count level')
    parser.add_argument('--mode', type=str, choices=['image', 'bert', 'gptj'], default='image',
                        help='Type of request to send (image, bert, or gptj)')
    parser.add_argument('--max-new-tokens', type=int, default=128,
                        help='Maximum number of new tokens to generate (for GPT-J only)')
    parser.add_argument('--output', type=str, default="latency_results.json",
                        help='Output file for latency results (JSON format)')
    parser.add_argument('--request-data-output', type=str, default="request_data.csv",
                        help='Output file for detailed request data')
    parser.add_argument('--output-format', type=str, choices=['json', 'csv'], default='csv',
                        help='Format for request data output (json or csv)')
    
    args = parser.parse_args()
    
    # Make sure image exists if in image mode
    if args.mode == 'image' and not os.path.exists(args.image):
        print(f"Please ensure '{args.image}' exists in the current directory")
        exit(1)

    print(f"Testing with URL: {args.url}")
    print(f"Mode: {args.mode}")
    if args.host:
        print(f"Using Host header: {args.host}")

    # Clear request times and data before starting
    request_times.clear()
    request_data.clear()

    # Determine which test mode to run
    if args.request_counts_file:
        print("\nRunning request counts-based test")
        timestamp_str = datetime.fromtimestamp(time.time()).strftime('%Y-%m-%d %H:%M:%S.%f')
        print(timestamp_str)
        run_request_counts(
            request_counts_file=args.request_counts_file,
            duration_per_stage=args.request_counts_duration,
            url=args.url,
            host_header=args.host,
            mode=args.mode,
            image_path=args.image,
            max_new_tokens=args.max_new_tokens
        )
    else:
        # Legacy modes
        # print("\n1. Sequential requests with delay")
        # send_multiple_requests(num_requests=args.sequential, delay=args.delay, 
        #                        url=args.url, host_header=args.host, mode=args.mode, image_path=args.image)
        # time.sleep(2)  # Wait between tests
        
        # print("\n2. Burst of concurrent requests")
        # send_concurrent_requests(num_requests=args.concurrent, url=args.url, host_header=args.host,
        #                         mode=args.mode, image_path=args.image)
        # time.sleep(2)  # Wait between tests
        
        print("\n3. Rate-limited requests")
        timestamp_str = datetime.fromtimestamp(time.time()).strftime('%Y-%m-%d %H:%M:%S.%f')
        print(timestamp_str)
        send_rate_limited_requests(requests_per_second=args.rate, duration_seconds=args.duration,
                                url=args.url, host_header=args.host, mode=args.mode, image_path=args.image,
                                max_new_tokens=args.max_new_tokens)
    
    # Calculate and display latency statistics
    latency_stats = calculate_latency_stats()
    
    print("\n===== Latency Statistics =====")
    print(f"Total Requests: {latency_stats['total_requests']}")
    print(f"Average Latency: {latency_stats['avg_latency']:.3f} seconds")
    print(f"P90 Latency: {latency_stats['p90_latency']:.3f} seconds")
    print(f"P95 Latency: {latency_stats['p95_latency']:.3f} seconds")
    print(f"P99 Latency: {latency_stats['p99_latency']:.3f} seconds")
    print(f"Min Latency: {latency_stats['min_latency']:.3f} seconds")
    print(f"Max Latency: {latency_stats['max_latency']:.3f} seconds")
    
    # Save results to file
    save_latency_results(latency_stats, args.output)
    
    # Save detailed request data
    save_request_data(args.request_data_output, args.output_format)