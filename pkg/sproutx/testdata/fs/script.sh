#!/bin/bash

# @block:variables
export API_URL="https://api.example.com"
export MAX_RETRIES=3
export TIMEOUT=30
# @endblock:variables

# @block:functions
function log_info() {
    echo "[INFO] $(date): $1"
}

function log_error() {
    echo "[ERROR] $(date): $1" >&2
}

function retry_command() {
    local command="$1"
    local max_attempts="$2"
    local attempt=1

    while [ $attempt -le $max_attempts ]; do
        log_info "Attempt $attempt of $max_attempts: $command"
        if eval "$command"; then
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done

    log_error "Command failed after $max_attempts attempts: $command"
    return 1
}
# @endblock

# @block:setup
# Create necessary directories
mkdir -p /tmp/myapp/{logs,data,config}

# Set permissions
chmod 755 /tmp/myapp
chmod 700 /tmp/myapp/logs
# @endblock:setup

# Main execution
log_info "Starting application setup"

# @block:main
if [ $# -eq 0 ]; then
    log_error "No arguments provided"
    echo "Usage: $0 <environment>"
    exit 1
fi

ENVIRONMENT="$1"
CONFIG_FILE="/tmp/myapp/config/${ENVIRONMENT}.conf"

# @block:config
cat > "$CONFIG_FILE" << EOF
# Configuration for $ENVIRONMENT
api_url=$API_URL
max_retries=$MAX_RETRIES
timeout=$TIMEOUT
log_level=INFO
EOF
# @endblock:config

log_info "Configuration written to $CONFIG_FILE"
# @endblock:main

# @block:cleanup
# Function to cleanup on exit
cleanup() {
    log_info "Cleaning up temporary files"
    rm -rf /tmp/myapp/data/*
    log_info "Cleanup complete"
}

# Register cleanup function
trap cleanup EXIT
# @endblock
