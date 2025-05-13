#!/bin/bash
# uninstall_fast_gshare.sh
# Script to uninstall FaST-GShare FaSTPod components
# Usage: ./uninstall_fast_gshare.sh [repo_dir]
#   repo_dir: Directory of FaST-GShare repository (default: ./FaST-GShare)

set -e  # Exit immediately if a command exits with non-zero status

# Default repository directory
REPO_DIR=${1:-"./FaST-GShare"}
LOGFILE="fast_gshare_uninstall_log.txt"

# Function for logging
log() {
  local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
  echo "[$timestamp] $1" | tee -a "$LOGFILE"
}

# Check if repository directory exists
check_repo_dir() {
  log "Checking if FaST-GShare repository directory exists at '$REPO_DIR'..."
  
  if [ ! -d "$REPO_DIR" ]; then
    log "ERROR: Directory '$REPO_DIR' does not exist."
    log "Please provide the correct repository path."
    return 1
  fi
  
  log "Repository directory '$REPO_DIR' exists."
  return 0
}

# Check if Kubernetes is available
check_kubernetes() {
  log "Checking if kubectl is installed and a cluster is accessible..."
  
  if ! command -v kubectl &> /dev/null; then
    log "ERROR: kubectl command not found. Please install kubectl first."
    return 1
  fi
  
  if ! kubectl cluster-info &> /dev/null; then
    log "ERROR: Unable to connect to Kubernetes cluster. Please check your configuration."
    return 1
  fi
  
  log "Kubernetes cluster is accessible."
  return 0
}

# Check if FaSTPod resources exist
check_fastpod_resources() {
  log "Checking if FaSTPod resources exist..."
  
  if ! kubectl get namespace fast-gshare &> /dev/null; then
    log "WARNING: 'fast-gshare' namespace does not exist. FaSTPod may not be installed."
    read -p "Continue with uninstallation anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      log "Aborting uninstallation."
      return 1
    fi
  else
    log "Found 'fast-gshare' namespace. Listing resources before uninstallation:"
    kubectl get all -n fast-gshare || true
    kubectl get fastpods -n fast-gshare 2>/dev/null || log "No FaSTPods found in fast-gshare namespace."
  fi
  
  return 0
}

# Uninstall FaSTPod deployment
uninstall_fastpod() {
  log "Uninstalling FaSTPod deployment..."
  
  local uninstall_script="$REPO_DIR/yaml/fastgshare/clean_deploy_ctr_mgr_node_daemon.sh"
  
  if [ -f "$uninstall_script" ]; then
    log "Running uninstallation script: $uninstall_script"
    # Save current directory
    local current_dir=$(pwd)
    cd "$REPO_DIR"
    bash ./yaml/fastgshare/clean_deploy_ctr_mgr_node_daemon.sh
    # Return to original directory
    cd "$current_dir"
    log "Successfully ran uninstallation script."
  else
    log "ERROR: Uninstallation script not found at expected path: $uninstall_script"
    log "Repository structure may have changed. Please check the repository."
    return 1
  fi
  
  return 0
}

# Clean up test resources
cleanup_test_resources() {
  log "Checking for and removing any test resources..."
  
  local test_yaml="$REPO_DIR/yaml/fastpod/testfastpod.yaml"
  
  if [ -f "$test_yaml" ] && kubectl get namespace fast-gshare &> /dev/null; then
    log "Removing test FaSTPod resources if they exist..."
    kubectl delete -f "$test_yaml" --ignore-not-found=true
    log "Test resources cleanup completed."
  else
    log "Skipping test resources cleanup (test file not found or namespace does not exist)."
  fi
  
  return 0
}

# Verify uninstallation
verify_uninstallation() {
  log "Verifying uninstallation..."
  
  if kubectl get namespace fast-gshare &> /dev/null; then
    log "WARNING: 'fast-gshare' namespace still exists after uninstallation."
    log "Listing remaining resources in the namespace:"
    kubectl get all -n fast-gshare || true
    
    read -p "Do you want to force delete the 'fast-gshare' namespace? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      log "Force deleting 'fast-gshare' namespace..."
      kubectl delete namespace fast-gshare --force --grace-period=0
      log "Force deletion initiated. This may take some time to complete."
    else
      log "Skipping force deletion. Some resources may still remain."
    fi
  else
    log "Uninstallation successfully removed the 'fast-gshare' namespace."
  fi
  
  # Check if CRD still exists
  if kubectl get crd fastpods.fastgshare.nvidia.com &> /dev/null; then
    log "WARNING: FaSTPod CRD still exists after uninstallation."
    
    read -p "Do you want to manually remove the FaSTPod CRD? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      log "Removing FaSTPod CRD..."
      kubectl delete crd fastpods.fastgshare.nvidia.com
      log "FaSTPod CRD removed."
    else
      log "Skipping CRD removal."
    fi
  else
    log "FaSTPod CRD successfully removed."
  fi
  
  return 0
}

# Main execution
log "Starting FaST-GShare FaSTPod uninstallation process..."

# Check prerequisites
if ! check_repo_dir; then
  log "ERROR: Repository directory check failed. Cannot proceed with uninstallation."
  exit 1
fi

if ! check_kubernetes; then
  log "ERROR: Kubernetes check failed. Cannot proceed with uninstallation."
  exit 1
fi

# Check for confirmation
echo "==============================================="
echo "WARNING: This will uninstall FaST-GShare FaSTPod components"
echo "from your Kubernetes cluster."
echo "==============================================="
read -p "Are you sure you want to continue with uninstallation? (y/n) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
  log "Uninstallation cancelled by user."
  exit 0
fi

# Check for existing resources
check_fastpod_resources

# Clean up test resources first
echo "Step 1: Cleaning up any test resources..."
cleanup_test_resources

# Uninstall FaSTPod components
echo "Step 2: Uninstalling FaSTPod components..."
if ! uninstall_fastpod; then
  log "ERROR: Failed to run uninstallation script."
  exit 1
fi

# Verify uninstallation
echo "Step 3: Verifying uninstallation..."
verify_uninstallation

log "FaSTPod uninstallation process completed."
echo "==============================================="
echo "FaST-GShare FaSTPod has been uninstalled"
echo "See uninstallation log for details: $LOGFILE"
echo "==============================================="

exit 0