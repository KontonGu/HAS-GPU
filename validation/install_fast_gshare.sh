#!/bin/bash
# install_fast_gshare.sh
# Script to install FaST-GShare FaSTPod components
# Usage: ./install_fast_gshare.sh [repo_dir]
#   repo_dir: Directory of FaST-GShare repository (default: ./FaST-GShare)

set -e  # Exit immediately if a command exits with non-zero status

# Default repository directory
REPO_DIR=${1:-"./FaST-GShare"}
LOGFILE="fast_gshare_install_log.txt"

# Function for logging
log() {
  local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
  echo "[$timestamp] $1" | tee -a "$LOGFILE"
}

# Copy libhas.so.1 to /fastpod/library
copy_libhas() {
  log "Checking if libhas.so.1 needs to be copied to /fastpod/library..."
  
  local libhas_path="$REPO_DIR/lib/libhas.so.1"
  
  # Check if the libhas.so.1 file exists in the repository
  if [ ! -f "$libhas_path" ]; then
    log "ERROR: libhas.so.1 file not found at expected path: $libhas_path"
    log "Repository structure may have changed. Please check the repository."
    return 1
  fi
  
  # Ask for confirmation before copying
  read -p "Do you want to copy libhas.so.1 to /fastpod/library? (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log "Skipping copying libhas.so.1 to /fastpod/library."
    return 0
  fi
  
  # Create the /fastpod/library directory if it doesn't exist
  if [ ! -d "/fastpod/library" ]; then
    log "Creating directory /fastpod/library..."
    sudo mkdir -p /fastpod/library
    if [ $? -ne 0 ]; then
      log "ERROR: Failed to create directory /fastpod/library. Please check permissions."
      return 1
    fi
  fi
  
  # Copy the file to /fastpod/library
  log "Copying $libhas_path to /fastpod/library..."
  sudo cp "$libhas_path" /fastpod/library/
  if [ $? -ne 0 ]; then
    log "ERROR: Failed to copy libhas.so.1 to /fastpod/library. Please check permissions."
    return 1
  fi
  
  log "Successfully copied libhas.so.1 to /fastpod/library."
  return 0
}


# Check if repository directory exists
check_repo_dir() {
  log "Checking if FaST-GShare repository directory exists at '$REPO_DIR'..."
  
  if [ ! -d "$REPO_DIR" ]; then
    log "ERROR: Directory '$REPO_DIR' does not exist."
    log "Please run setup_fast_gshare.sh first or provide the correct repository path."
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

# Install FaSTPod CRD
install_fastpod_crd() {
  log "Deploying FaSTPod CRD (Custom Resource Definition)..."
  
  local crd_path="$REPO_DIR/yaml/crds/fastpod_crd.yaml"
  
  if [ -f "$crd_path" ]; then
    kubectl apply -f "$crd_path"
    log "Successfully deployed FaSTPod CRD."
  else
    log "ERROR: FaSTPod CRD file not found at expected path: $crd_path"
    log "Repository structure may have changed. Please check the repository."
    return 1
  fi
  
  return 0
}

# Deploy FaSTPod Controller Manager and GPU Resource Configurator
deploy_controller_manager() {
  log "Deploying FaSTPod Controller Manager and GPU Resource Configurator..."
  
  local deploy_script="$REPO_DIR/yaml/fastgshare/apply_deploy_ctr_mgr_node_daemon.sh"
  
  if [ -f "$deploy_script" ]; then
    log "Running deployment script: $deploy_script"
    # Save current directory
    local current_dir=$(pwd)
    cd "$REPO_DIR"
    bash ./yaml/fastgshare/apply_deploy_ctr_mgr_node_daemon.sh
    # Return to original directory
    cd "$current_dir"
    log "Successfully deployed controller manager and GPU resource configurator."
  else
    log "ERROR: Deployment script not found at expected path: $deploy_script"
    log "Repository structure may have changed. Please check the repository."
    return 1
  fi
  
  return 0
}

# Test FaSTPod deployment
test_fastpod() {
  log "Testing FaSTPod deployment with example configuration..."
  
  read -p "Do you want to run the FaSTPod test example? (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log "Skipping FaSTPod test."
    return 0
  fi
  
  local test_yaml="$REPO_DIR/yaml/fastpod/testfastpod.yaml"
  
  if [ -f "$test_yaml" ]; then
    kubectl apply -f "$test_yaml"
    log "Test FaSTPod example deployed."
    
    echo "Checking deployed resources..."
    echo "FaSTPod Pods:"
    kubectl get pods -n fast-gshare
    echo "FaSTPod resources:"
    kubectl get fastpods -n fast-gshare
  else
    log "ERROR: Test FaSTPod file not found at expected path: $test_yaml"
    log "Repository structure may have changed. Please check the repository."
    return 1
  fi
  
  return 0
}

# Check and create kube-config if needed
check_kubeconfig() {
  log "Checking if kube-config exists in kube-system namespace..."
  
  if ! kubectl get configmap kube-config -n kube-system &> /dev/null; then
    log "kube-config not found in kube-system namespace."
    read -p "Do you want to create the kube-config configmap? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      if [ -f "$HOME/.kube/config" ]; then
        kubectl create configmap kube-config -n kube-system --from-file=$HOME/.kube/config
        log "Created kube-config configmap from $HOME/.kube/config"
      else
        log "ERROR: Kubeconfig file not found at $HOME/.kube/config"
        log "Please specify the correct path to your kubeconfig file."
      fi
    else
      log "Skipping kube-config creation."
    fi
  else
    log "kube-config configmap already exists in kube-system namespace."
  fi
}

# Uninstall FaSTPod deployment
uninstall_fastpod() {
  log "Checking if you want to uninstall FaSTPod deployment..."
  
  read -p "Do you want to uninstall the FaSTPod deployment? (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log "Skipping uninstallation."
    return 0
  fi
  
  local uninstall_script="$REPO_DIR/yaml/fastgshare/clean_deploy_ctr_mgr_node_daemon.sh"
  
  if [ -f "$uninstall_script" ]; then
    log "Running uninstallation script: $uninstall_script"
    # Save current directory
    local current_dir=$(pwd)
    cd "$REPO_DIR"
    bash ./yaml/fastgshare/clean_deploy_ctr_mgr_node_daemon.sh
    # Return to original directory
    cd "$current_dir"
    log "Successfully uninstalled FaSTPod deployment."
  else
    log "ERROR: Uninstallation script not found at expected path: $uninstall_script"
    log "Repository structure may have changed. Please check the repository."
    return 1
  fi
  
  return 0
}

# Main execution
log "Starting FaST-GShare FaSTPod installation process..."

# Check prerequisites
if ! check_repo_dir; then
  log "ERROR: Repository directory check failed. Cannot proceed with installation."
  exit 1
fi

# Copy libhas.so.1 to /fastpod/library
echo "Step 1: Copying libhas.so.1 to /fastpod/library..."
if ! copy_libhas; then
  log "ERROR: Failed to copy libhas.so.1 to /fastpod/library. Cannot proceed with installation."
  exit 1
fi

if ! check_kubernetes; then
  log "ERROR: Kubernetes check failed. Cannot proceed with installation."
  exit 1
fi

# Install FaSTPod components
echo "==============================================="
echo "Installing FaST-GShare FaSTPod components..."
echo "==============================================="

# Step 2: Install FaSTPod CRD
echo "Step 2: Deploy FaSTPod CRD"
if ! install_fastpod_crd; then
  log "ERROR: Failed to install FaSTPod CRD."
  exit 1
fi

# Step 3: Deploy FaSTPod Controller Manager and GPU Resource Configurator
echo "Step 3: Deploy FaSTPod Controller Manager and GPU Resource Configurator"
if ! deploy_controller_manager; then
  log "ERROR: Failed to deploy controller manager."
  exit 1
fi

# Step 4: Test FaSTPod deployment
echo "Step 4: Test FaSTPod deployment"
test_fastpod

# Step 5: Check kubeconfig
echo "Step 5: Checking kube-config"
check_kubeconfig

log "FaSTPod installation process completed successfully."
echo "==============================================="
echo "FaST-GShare FaSTPod has been installed"
echo "See installation log for details: $LOGFILE"
echo "==============================================="

# Ask about uninstallation
echo "Do you want to uninstall the deployment?"
echo "This is useful for testing but would remove the installed components."
uninstall_fastpod

exit 0