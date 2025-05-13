#!/bin/bash
# setup_fast_gshare.sh
# Script to download and set up the FaST-GShare repository
# Usage: ./setup_fast_gshare.sh [install_dir]

set -e  # Exit immediately if a command exits with non-zero status

# Default installation directory
INSTALL_DIR=${1:-"./FaST-GShare"}
REPO_URL="https://github.com/KontonGu/FaST-GShare.git"  # FaST-GShare repository URL
LOGFILE="fast_gshare_setup_log.txt"
BRANCH="has"  # Branch to checkout

# Function for logging
log() {
  local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
  echo "[$timestamp] $1" | tee -a "$LOGFILE"
}

# Clone the repository
clone_repo() {
  log "Checking if directory '$INSTALL_DIR' already exists..."
  
  if [ -d "$INSTALL_DIR" ]; then
    log "Directory '$INSTALL_DIR' already exists."
    read -p "Do you want to remove and re-clone it? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      log "Removing existing directory '$INSTALL_DIR'..."
      rm -rf "$INSTALL_DIR"
      log "Cloning repository from $REPO_URL to $INSTALL_DIR..."
      git clone "$REPO_URL" "$INSTALL_DIR"
    else
      log "Updating existing repository..."
      cd "$INSTALL_DIR"
      git pull
      cd - > /dev/null
    fi
  else
    log "Cloning repository from $REPO_URL to $INSTALL_DIR..."
    git clone "$REPO_URL" "$INSTALL_DIR"
  fi
  
  log "Repository setup completed."
}

# Update gateway image in values.yaml
update_gateway_image() {
  log "Updating gateway image in values.yaml..."
  
  # Path to values.yaml
  local values_yaml_path="$INSTALL_DIR/chart/fastgshare/values.yaml"
  
  if [ -f "$values_yaml_path" ]; then
    log "Found values.yaml at $values_yaml_path"
    
    # Create backup of original file
    cp "$values_yaml_path" "${values_yaml_path}.bak"
    log "Created backup at ${values_yaml_path}.bak"
    
    # Use sed to replace the gateway image line
    # The pattern looks for lines containing 'image:' within the gateway section
    # and replaces with our specified image
    sed -i.tmp '/gateway:/,/basicAuthPlugin:/ s|image:.*|image: leslie233/gateway:has|' "$values_yaml_path"
    
    # Clean up temporary file created by sed
    rm -f "${values_yaml_path}.tmp"
    
    # Verify the change
    if grep -q "leslie233/gateway:has" "$values_yaml_path"; then
      log "Successfully updated gateway image to leslie233/gateway:has"
    else
      log "WARNING: Failed to update gateway image. Please update manually."
      log "The line that needs to be changed is in the gateway section of $values_yaml_path"
      log "Change the image line to: image: leslie233/gateway:has"
    fi
  else
    log "ERROR: values.yaml not found at expected path: $values_yaml_path"
    log "Repository structure may have changed. Please update the gateway image manually."
  fi
}

# Checkout specific branch
checkout_branch() {
  log "Checking out '$BRANCH' branch..."
  cd "$INSTALL_DIR"
  
  # Check if branch exists
  if git show-ref --verify --quiet "refs/remotes/origin/$BRANCH"; then
    # Checkout the branch
    git checkout "$BRANCH"
    log "Successfully checked out '$BRANCH' branch."
  else
    log "WARNING: Branch '$BRANCH' not found. Staying on default branch."
    log "Available branches:"
    git branch -a
  fi
  
  cd - > /dev/null
}

# Main execution
log "Starting FaST-GShare setup process..."
clone_repo
checkout_branch
# No longer update gateway image for now
#update_gateway_image



log "FaST-GShare setup process completed successfully."
echo "==============================================="
echo "FaST-GShare has been set up in: $INSTALL_DIR"
echo "To explore the repository: cd $INSTALL_DIR"
echo "See setup log for details: $LOGFILE"
echo "==============================================="