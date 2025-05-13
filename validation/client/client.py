# client.py
import json
import time
import argparse
import threading
from concurrent.futures import ThreadPoolExecutor
import numpy as np
import requests
import os
import heapq
from datetime import datetime
from collections import defaultdict

class PerformanceStats:
    def __init__(self):
        self.latencies = []
        self.start_time = float('inf')
        self.end_time = 0
        self.lock = threading.Lock()
        self.request_count = 0

    def add_request(self, start: float, end: float):
        with self.lock:
            self.latencies.append(end - start)
            self.start_time = min(self.start_time, start)
            self.end_time = max(self.end_time, end)
            self.request_count += 1

    def calculate_stats(self):
        if not self.latencies:
            return {}

        latencies = sorted(self.latencies)
        total_time = self.end_time - self.start_time
        return {
            'throughput': self.request_count / total_time if total_time > 0 else 0,
            'p95': np.percentile(latencies, 95),
            'p99': np.percentile(latencies, 99),
            'avg': np.mean(latencies),
            'min': np.min(latencies),
            'max': np.max(latencies),
            'total_requests': self.request_count,
            'total_time': total_time
        }

class Client:
    def __init__(self, concurrency: int = 16, config_file: str = None):
        self.concurrency = concurrency
        self.stats = PerformanceStats()
        self.app_configs = {}
        self.app_stats_files = {}
        self.app_concurrency_data = {}
        if config_file:
            self._load_config(config_file)

    def _load_config(self, config_file: str):
        try:
            with open(config_file, 'r') as f:
                config_data = json.load(f)
                self.app_configs = config_data.get('apps', {})
                print(f"Loaded configurations for apps: {list(self.app_configs.keys())}")
        except Exception as e:
            print(f"Error loading config file: {e}")
            raise

    def _load_app_stats(self, app_stats_files: dict):
        """
        Load app stats files containing concurrency history data
        
        Args:
            app_stats_files: Dictionary mapping app names to stats file paths
        """
        self.app_stats_files = app_stats_files
        self.app_concurrency_data = {}
        
        for app_name, file_path in app_stats_files.items():
            if app_name not in self.app_configs:
                print(f"Warning: App '{app_name}' has stats file but no configuration")
                continue
                
            try:
                print(f"Loading stats for app '{app_name}' from {file_path}")
                with open(file_path, 'r') as f:
                    stats_data = json.load(f)
                    
                # Extract concurrency history
                if 'concurrency_history' in stats_data:
                    # Format: [(timestamp, concurrency_level), ...]
                    self.app_concurrency_data[app_name] = stats_data['concurrency_history']
                    print(f"Loaded {len(self.app_concurrency_data[app_name])} concurrency data points for '{app_name}'")
                else:
                    print(f"Warning: No concurrency history found in stats file for '{app_name}'")
            except Exception as e:
                print(f"Error loading stats file for '{app_name}': {e}")

    def _send_request(self, app_name: str, timestamp: float = None, custom_payload=None, verbose=False):
        """
        Send a request for a specific app
        
        Args:
            app_name: Name of the app to send request to
            timestamp: Optional timestamp for logging
            custom_payload: Optional custom payload for the request
            verbose: Whether to print verbose output
        """
        if app_name not in self.app_configs:
            print(f"App '{app_name}' not found in configuration")
            return

        app_config = self.app_configs.get(app_name)
        base_url = app_config.get('base_url')
        if not base_url:
            print(f"Base URL not configured for app '{app_name}'")
            return

        # Build complete URL with path
        path = app_config.get('path', '').lstrip('/')
        url = f"{base_url.rstrip('/')}/{path}" if path else base_url

        headers = app_config.get('headers', {})
        timeout = app_config.get('timeout', 10)
        input_file = app_config.get('input_file')

        if not input_file:
            print(f"Input file not configured for app '{app_name}'")
            return

        start_time = time.time()
        try:
            # Determine app type based on app name or input file
            app_type = "image"  # Default
            if app_name == "bert" or "bert" in app_name.lower():
                app_type = "bert"
            elif app_name == "gpt-j" or "gpt" in app_name.lower():
                app_type = "gptj"
                
            # Handle different types of requests based on app type
            if app_type == "bert":
                # BERT question-answering request
                sample = {
                    "question": "What is the capital of France?",
                    "context": "Paris is the capital and largest city of France. It is situated on the river Seine."
                }
                
                if custom_payload:
                    sample = custom_payload
                
                headers["Content-Type"] = "application/json"
                response = requests.post(
                    url=url,
                    json=sample,
                    headers=headers,
                    timeout=timeout
                )
            elif app_type == "gptj":
                # GPT-J text generation request
                payload = {
                    "texts": ["Write a short story about a robot learning to paint:"]
                }
                
                if custom_payload:
                    payload = custom_payload
                
                headers["Content-Type"] = "application/json"
                response = requests.post(
                    url=url,
                    json=payload,
                    headers=headers,
                    timeout=timeout
                )
            else:
                # Image-based request
                # Determine if we're using KServe or local endpoint
                if "kserve" in url or "v1/models" in url:
                    # KServe expects a different format
                    with open(input_file, 'rb') as img_file:
                        img_bytes = img_file.read()
                        
                    # For KServe, we need to send binary data
                    headers["Content-Type"] = "application/octet-stream"
                    response = requests.post(
                        url=url,
                        data=img_bytes,
                        headers=headers,
                        timeout=timeout
                    )
                else:
                    # Original format for local endpoint
                    with open(input_file, 'rb') as f:
                        files = {'payload': f}
                        response = requests.post(
                            url=url,
                            files=files,
                            headers=headers,
                            timeout=timeout
                        )
            
            response.raise_for_status()
            
            # Print timestamp for request
            if timestamp:
                print(f"Request to {app_name} sent at timestamp {timestamp:.3f}")
            else:
                print(f"Request to {app_name} sent at {time.strftime('%H:%M:%S')}")
                
        except Exception as e:
            print(f"Request to {url} failed: {str(e)}")
        else:
            end_time = time.time()
            self.stats.add_request(start_time, end_time)
            
            # Print response summary if verbose
            if verbose:
                try:
                    if response.headers.get('Content-Type', '').startswith('application/json'):
                        print(f"Response: {response.json()}")
                    else:
                        print(f"Response status: {response.status_code}")
                except:
                    print(f"Response status: {response.status_code}")

    def run(self, input_file: str):
        with ThreadPoolExecutor(max_workers=self.concurrency) as executor:
            with open(input_file, 'r') as f:
                for line in f:
                    try:
                        request = json.loads(line.strip())
                        executor.submit(self._send_request, request.get('app'))
                    except json.JSONDecodeError:
                        print(f"Invalid JSON: {line.strip()}")

    def run_trace_based(self, time_scale: float = 1.0, start_offset: float = 0.0, max_duration: float = None, verbose: bool = False):
        """
        Run requests based on trace data, following the concurrency patterns in the app stats files
        
        Args:
            time_scale: Scale factor for time (e.g., 0.5 = run twice as fast as original trace)
            start_offset: Offset to apply to all timestamps (in seconds)
            max_duration: Maximum duration to run (in seconds)
            verbose: Whether to print verbose output
        """
        if not self.app_concurrency_data:
            print("No app concurrency data loaded. Use load_app_stats() first.")
            return
            
        print(f"\nStarting trace-based load test with time scale {time_scale}x...")
        print(f"Apps with concurrency data: {list(self.app_concurrency_data.keys())}")
        
        # Create a timeline of events by merging all app concurrency histories
        timeline = []
        
        # Process each app's concurrency history
        for app_name, history in self.app_concurrency_data.items():
            prev_concurrency = 0
            
            for i, (timestamp, concurrency) in enumerate(history):
                # Apply time scaling and offset
                adjusted_timestamp = start_offset + (timestamp * time_scale)
                
                # Calculate delta (change in concurrency)
                delta = concurrency - prev_concurrency
                
                if delta != 0:
                    # Add event to timeline
                    timeline.append((adjusted_timestamp, app_name, delta))
                    
                prev_concurrency = concurrency
                
                # Check if we've reached max duration
                if max_duration and (adjusted_timestamp - start_offset) > max_duration:
                    break
        
        # Sort timeline by timestamp
        timeline.sort()
        
        if not timeline:
            print("No events in timeline. Check your app stats files.")
            return
            
        print(f"Created timeline with {len(timeline)} events")
        print(f"Timeline spans from {timeline[0][0]:.3f}s to {timeline[-1][0]:.3f}s")
        
        # Track current concurrency for each app
        current_concurrency = defaultdict(int)
        
        # Start time for the test
        start_time = time.time()
        
        # Process events in chronological order
        prev_event_time = timeline[0][0]
        
        for event_time, app_name, delta in timeline:
            # Wait until it's time for this event
            wait_time = (event_time - prev_event_time)
            if wait_time > 0:
                time.sleep(wait_time)
                
            # Update concurrency for this app
            current_concurrency[app_name] += delta
            
            # Log the event
            elapsed = time.time() - start_time
            print(f"[{elapsed:.3f}s] App '{app_name}' concurrency changed by {delta} to {current_concurrency[app_name]}")
            
            # If concurrency increased, send additional requests
            if delta > 0:
                # Launch delta new requests
                for _ in range(delta):
                    threading.Thread(
                        target=self._send_request,
                        kwargs={
                            'app_name': app_name,
                            'timestamp': event_time,
                            'verbose': verbose
                        }
                    ).start()
            
            prev_event_time = event_time
            
            # Check if we've reached max duration
            if max_duration and (elapsed > max_duration):
                print(f"Reached maximum duration of {max_duration}s, stopping")
                break
        
        # Wait for any remaining requests to complete
        time.sleep(2)
        
        # Return stats
        return self.stats.calculate_stats()

    def run_request_counts(self, json_file: str, duration: float = 2.0, url: str = None, host_header: str = None, 
                          image_path: str = "car.jpg", mode: str = "image", max_new_tokens: int = 128, verbose: bool = False):
        """
        Run requests based on processed_request_counts from a JSON file
        
        Args:
            json_file: Path to JSON file containing processed_request_counts
            duration: Duration in seconds to maintain each request count level
            url: Direct URL to send requests to
            host_header: Optional host header for KServe
            image_path: Path to the image file for image-based requests
            mode: Type of request ("image", "bert", or "gptj")
            max_new_tokens: Maximum number of new tokens to generate (for GPT-J only)
            verbose: Whether to print verbose output
        """
        if not url:
            print(f"Error: URL is required for run_request_counts mode")
            return
            
        try:
            # Load JSON data
            with open(json_file, 'r') as f:
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
            print(f"Each stage will run for {duration} seconds")
            print(f"URL: {url}")
            if host_header:
                print(f"Host header: {host_header}")
            print(f"Mode: {mode}")
            
            # Track active threads
            active_threads = []
            stop_event = threading.Event()
            
            # Start time for the test
            start_time = time.time()
            
            # Process each request count
            for i, count in enumerate(request_counts):
                count = int(count)  # Ensure count is an integer
                
                # Log the stage
                elapsed = time.time() - start_time
                print(f"[{elapsed:.3f}s] Stage {i+1}/{len(request_counts)}: Setting request rate to {count} requests/second")
                
                # Clear previous stop event and create a new one
                stop_event.clear()
                stop_event = threading.Event()
                
                # Clear active threads list
                active_threads = []
                
                # Calculate interval between requests to achieve desired rate
                if count > 0:
                    interval = 1.0 / count
                else:
                    interval = 0
                
                # Create and start request threads
                for _ in range(count):
                    thread = threading.Thread(
                        target=self._send_request_simple,
                        kwargs={
                            'url': url,
                            'host_header': host_header,
                            'image_path': image_path,
                            'mode': mode,
                            'max_new_tokens': max_new_tokens,
                            'interval': interval,
                            'stop_event': stop_event,
                            'verbose': verbose
                        }
                    )
                    thread.daemon = True
                    thread.start()
                    active_threads.append(thread)
                
                # Wait for the specified duration
                time.sleep(duration)
                
                # Signal threads to stop
                stop_event.set()
                
                # Wait for threads to finish (with timeout)
                for thread in active_threads:
                    thread.join(timeout=0.5)
            
            # Wait for any remaining requests to complete
            time.sleep(1)
            
            # Return stats
            return self.stats.calculate_stats()
            
        except FileNotFoundError:
            print(f"Error: JSON file '{json_file}' not found.")
        except json.JSONDecodeError:
            print(f"Error: '{json_file}' is not a valid JSON file.")
        except Exception as e:
            print(f"Error processing request counts: {str(e)}")
    
    def _send_request_simple(self, url: str, host_header: str = None, image_path: str = "car.jpg", 
                           mode: str = "image", max_new_tokens: int = 128,
                           interval: float = 0, stop_event: threading.Event = None, verbose: bool = False):
        """
        Send a simple request to the specified URL
        
        Args:
            url: URL to send the request to
            host_header: Optional host header for KServe
            image_path: Path to the image file for image-based requests
            mode: Type of request ("image", "bert", or "gptj")
            max_new_tokens: Maximum number of new tokens to generate (for GPT-J only)
            interval: Interval between requests in seconds
            stop_event: Event to signal when to stop sending requests
            verbose: Whether to print verbose output
        """
        while not stop_event or not stop_event.is_set():
            start_time = time.time()
            
            try:
                # Set up headers
                headers = {"Accept": "application/json"}
                if host_header:
                    headers["Host"] = host_header
                
                # Handle different types of requests based on mode
                if mode == "bert":
                    # BERT question-answering request
                    sample = {
                        "question": "What is the capital of France?",
                        "context": "Paris is the capital and largest city of France. It is situated on the river Seine."
                    }
                    
                    headers["Content-Type"] = "application/json"
                    response = requests.post(
                        url=url,
                        json=sample,
                        headers=headers,
                        timeout=10
                    )
                elif mode == "gptj":
                    # GPT-J text generation request
                    payload = {
                        "texts": ["Write a short story about a robot learning to paint:"],
                        "max_new_tokens": max_new_tokens
                    }
                    
                    headers["Content-Type"] = "application/json"
                    response = requests.post(
                        url=url,
                        json=payload,
                        headers=headers,
                        timeout=10
                    )
                else:
                    # Image-based request (default)
                    # Determine if we're using KServe or local endpoint
                    if "kserve" in url or "v1/models" in url:
                        # KServe expects a different format
                        with open(image_path, 'rb') as img_file:
                            img_bytes = img_file.read()
                            
                        # For KServe, we need to send binary data
                        headers["Content-Type"] = "application/octet-stream"
                        response = requests.post(
                            url=url,
                            data=img_bytes,
                            headers=headers,
                            timeout=10
                        )
                    else:
                        # Original format for local endpoint
                        with open(image_path, 'rb') as f:
                            files = {'payload': f}
                            response = requests.post(
                                url=url,
                                files=files,
                                headers=headers,
                                timeout=10
                            )
                
                response.raise_for_status()
                
                # Print response summary if verbose
                if verbose:
                    print(f"Request sent at {time.strftime('%H:%M:%S')} - Status: {response.status_code}")
                    
            except Exception as e:
                if verbose:
                    print(f"Request to {url} failed: {str(e)}")
            else:
                end_time = time.time()
                self.stats.add_request(start_time, end_time)
            
            # If no stop event or interval, just send once and return
            if not stop_event or interval <= 0:
                break
                
            # Calculate how long to sleep to maintain the interval
            elapsed = time.time() - start_time
            sleep_time = max(0, interval - elapsed)
            
            # Sleep for the remaining time or until stopped
            if stop_event.wait(timeout=sleep_time):
                # If event is set during wait, break the loop
                break

def main():
    parser = argparse.ArgumentParser(description='Load testing client with app configurations')
    parser.add_argument('--input', help='Input file with requests (each line is a JSON object)')
    parser.add_argument('--config', help='JSON configuration file for app settings')
    parser.add_argument('--concurrency', type=int, default=16,
                        help='Number of concurrent workers')
    parser.add_argument('--trace-mode', action='store_true',
                        help='Run in trace-based mode using app stats files')
    parser.add_argument('--request-counts-mode', action='store_true',
                        help='Run using request counts from a JSON file')
    parser.add_argument('--request-counts-file', 
                        help='JSON file containing processed_request_counts')
    parser.add_argument('--request-counts-duration', type=float, default=2.0,
                        help='Duration in seconds for each request count level')
    parser.add_argument('--url', 
                        help='Direct URL to send requests to')
    parser.add_argument('--host-header', 
                        help='Host header for KServe requests')
    parser.add_argument('--image-path', default='car.jpg',
                        help='Path to the image file for image-based requests')
    parser.add_argument('--mode', type=str, choices=['image', 'bert', 'gptj'], default='image',
                        help='Type of request to send (image, bert, or gptj)')
    parser.add_argument('--max-new-tokens', type=int, default=128,
                        help='Maximum number of new tokens to generate (for GPT-J only)')
    parser.add_argument('--app-name', 
                        help='Name of the app to send requests to in request-counts mode (only used with --config)')
    parser.add_argument('--app-stats', nargs='+',
                        help='App stats files in format app_name:file_path')
    parser.add_argument('--time-scale', type=float, default=1.0,
                        help='Time scale factor for trace-based mode')
    parser.add_argument('--max-duration', type=float,
                        help='Maximum duration to run in seconds')
    parser.add_argument('--verbose', action='store_true',
                        help='Print verbose output')
    args = parser.parse_args()

    # Create client with or without config
    if args.config:
        client = Client(concurrency=args.concurrency, config_file=args.config)
    else:
        client = Client(concurrency=args.concurrency)

    if args.request_counts_mode:
        if not args.request_counts_file:
            print("Error: --request-counts-file is required in request-counts mode")
            return
            
        if not args.url and not args.app_name:
            print("Error: Either --url or --app-name with --config is required in request-counts mode")
            return
            
        # Run request counts-based test
        print(f"\nStarting request counts-based load test...")
        start = time.time()
        
        # If URL is provided directly, use it instead of app configuration
        if args.url:
            stats = client.run_request_counts(
                json_file=args.request_counts_file,
                duration=args.request_counts_duration,
                url=args.url,
                host_header=args.host_header,
                image_path=args.image_path,
                mode=args.mode,
                max_new_tokens=args.max_new_tokens,
                verbose=args.verbose
            )
        else:
            # Use app configuration from config file
            stats = client.run_request_counts(
                json_file=args.request_counts_file,
                duration=args.request_counts_duration,
                app_name=args.app_name,
                verbose=args.verbose
            )
            
        duration = time.time() - start
    elif args.trace_mode:
        if not args.app_stats:
            print("Error: --app-stats is required in trace mode")
            return
            
        # Parse app stats files
        app_stats_files = {}
        for app_stat in args.app_stats:
            try:
                app_name, file_path = app_stat.split(':', 1)
                app_stats_files[app_name] = file_path
            except ValueError:
                print(f"Error: Invalid format for app stats '{app_stat}'. Use format 'app_name:file_path'")
                return
        
        # Load app stats
        client._load_app_stats(app_stats_files)
        
        # Run trace-based test
        print(f"\nStarting trace-based load test with time scale {args.time_scale}x...")
        start = time.time()
        stats = client.run_trace_based(
            time_scale=args.time_scale,
            max_duration=args.max_duration,
            verbose=args.verbose
        )
        duration = time.time() - start
    elif args.input:
        print(f"\nStarting load test with {args.concurrency} workers...")
        start = time.time()
        client.run(args.input)
        duration = time.time() - start
        stats = client.stats.calculate_stats()
    else:
        parser.print_help()
        return

    print("\n=== Load Test Results ===")
    print(f"Total requests processed: {stats.get('total_requests', 0)}")
    print(f"Total test time: {stats.get('total_time', 0):.2f} seconds")
    print(f"Throughput: {stats.get('throughput', 0):.2f} requests/second")
    print(f"Average latency: {stats.get('avg', 0):.4f} s")
    print(f"P95 latency: {stats.get('p95', 0):.4f} s")
    print(f"P99 latency: {stats.get('p99', 0):.4f} s")
    print(f"Min latency: {stats.get('min', 0):.4f} s")
    print(f"Max latency: {stats.get('max', 0):.4f} s")
    print(f"Client execution time: {duration:.2f} seconds")

if __name__ == '__main__':
    main()