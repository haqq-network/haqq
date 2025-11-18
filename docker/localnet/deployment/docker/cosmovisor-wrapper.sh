#!/bin/bash

# Signal handler for graceful shutdown
cleanup() {
    echo "Received signal, shutting down cosmovisor gracefully..."
    if [ -n "$COSMOVISOR_PID" ]; then
        kill -TERM "$COSMOVISOR_PID" 2>/dev/null || true
        wait "$COSMOVISOR_PID" 2>/dev/null || true
    fi
    exit 0
}

# Function to check and update symlink if needed
check_and_update_symlink() {
    local daemon_home="${DAEMON_HOME:-/app/.haqqd}"
    local daemon_name="${DAEMON_NAME:-haqqd}"
    local upgrade_info_file="$daemon_home/data/upgrade-info.json"
    local current_link="$daemon_home/cosmovisor/current"
    
    # Check if upgrade-info.json exists (indicates upgrade is pending or just happened)
    if [ -f "$upgrade_info_file" ]; then
        # Read the upgrade name from upgrade-info.json
        if command -v jq >/dev/null 2>&1; then
            local upgrade_name=$(jq -r '.name' "$upgrade_info_file" 2>/dev/null || echo "")
            if [ -n "$upgrade_name" ] && [ "$upgrade_name" != "null" ]; then
                echo "Upgrade detected: $upgrade_name"
                
                # Expected path for the upgrade directory (cosmovisor uses directory symlinks)
                local expected_dir="$daemon_home/cosmovisor/upgrades/$upgrade_name"
                local expected_binary="$expected_dir/bin/$daemon_name"
                
                # Check if the upgrade directory exists
                if [ ! -d "$expected_dir" ]; then
                    echo "Warning: Upgrade directory not found at $expected_dir"
                    return 1
                fi
                
                # Check if the upgrade binary exists
                if [ ! -f "$expected_binary" ]; then
                    echo "Warning: Upgrade binary not found at $expected_binary"
                    return 1
                fi
                
                # Check if the current symlink points to the upgrade version directory
                # The 'current' symlink should point to the version directory, not the binary
                if [ -L "$current_link" ]; then
                    local current_target=$(readlink "$current_link" 2>/dev/null || echo "")
                    # Resolve relative symlinks
                    local current_resolved=$(readlink -f "$current_link" 2>/dev/null || echo "")
                    local expected_resolved=$(readlink -f "$expected_dir" 2>/dev/null || echo "")
                    
                    # Check if symlink points to the correct upgrade directory
                    # Only compare if both resolved paths are non-empty
                    if [ -z "$current_resolved" ] || [ -z "$expected_resolved" ] || [ "$current_resolved" != "$expected_resolved" ]; then
                        echo "Symlink needs update. Current: $current_target -> $current_resolved, Expected: $expected_dir -> $expected_resolved"
                        # Try to update the symlink ourselves as fallback
                        echo "Attempting to update symlink to point to upgrade directory..."
                        # Remove old symlink if it exists
                        rm -f "$current_link" 2>/dev/null || true
                        # Create new symlink pointing to the upgrade directory
                        ln -sfn "$expected_dir" "$current_link" 2>/dev/null && echo "Symlink updated successfully" || echo "Failed to update symlink"
                        # Verify the update
                        local new_resolved=$(readlink -f "$current_link" 2>/dev/null || echo "")
                        if [ -n "$new_resolved" ] && [ -n "$expected_resolved" ] && [ "$new_resolved" = "$expected_resolved" ]; then
                            echo "Symlink successfully updated to point to upgrade directory: $upgrade_name"
                            return 0
                        else
                            echo "Symlink update verification failed. New target: $new_resolved, Expected: $expected_resolved"
                            return 1
                        fi
                    else
                        echo "Symlink is correctly pointing to upgrade directory: $upgrade_name"
                        return 0
                    fi
                else
                    echo "Current symlink does not exist, creating it..."
                    # Create the symlink pointing to the upgrade directory
                    ln -sfn "$expected_dir" "$current_link" 2>/dev/null && echo "Symlink created successfully" || echo "Failed to create symlink"
                    # Verify the creation
                    if [ -L "$current_link" ]; then
                        local new_resolved=$(readlink -f "$current_link" 2>/dev/null || echo "")
                        local expected_resolved=$(readlink -f "$expected_dir" 2>/dev/null || echo "")
                        if [ -n "$new_resolved" ] && [ -n "$expected_resolved" ] && [ "$new_resolved" = "$expected_resolved" ]; then
                            echo "Symlink successfully created and verified: $upgrade_name"
                            return 0
                        else
                            echo "Symlink creation verification failed. New target: $new_resolved, Expected: $expected_resolved"
                            return 1
                        fi
                    else
                        echo "Failed to create symlink"
                        return 1
                    fi
                fi
            fi
        else
            echo "Warning: jq not found, cannot parse upgrade-info.json"
        fi
    fi
    
    return 0
}

# Set up signal handlers
trap cleanup SIGTERM SIGINT

# Main loop to keep cosmovisor running
while true; do
    echo "Starting cosmovisor..."
    
    # Start cosmovisor in the background so we can track its PID
    # Build command arguments
    cmd_args=(
        "run" "start"
        "--pruning=nothing"
        "--log_level" "${LOGLEVEL:-info}"
        "--minimum-gas-prices=0.0001aISLM"
        "--json-rpc.api" "eth,txpool,personal,net,debug,web3"
        "--json-rpc.enable" "true"
        "--keyring-backend" "${KEYRING:-test}"
        "--chain-id" "${CHAINID:-haqq_121799-1}"
        "--home" "${DAEMON_HOME:-/app/.haqqd}"
    )
    
    # Add TRACE flag if set and non-empty
    if [ -n "${TRACE:-}" ]; then
        cmd_args+=("$TRACE")
    fi
    
    cosmovisor "${cmd_args[@]}" &
    
    COSMOVISOR_PID=$!
    
    # Wait for cosmovisor to exit
    set +e  # Don't exit on error, we want to handle it
    wait "$COSMOVISOR_PID"
    EXIT_CODE=$?
    set -e
    
    # Clear PID since process has exited
    COSMOVISOR_PID=""
    
    echo "Cosmovisor exited with code $EXIT_CODE"
    
    # Check if we should exit (received termination signal)
    # Exit code 143 = SIGTERM, 130 = SIGINT, 128+signal = other signals
    if [ "$EXIT_CODE" -eq 143 ] || [ "$EXIT_CODE" -eq 130 ] || [ "$EXIT_CODE" -ge 128 ]; then
        echo "Cosmovisor was terminated with signal, exiting wrapper..."
        exit 0
    fi
    
    # Check if an upgrade just happened and ensure symlink is updated
    echo "Checking upgrade status..."
    max_wait=30  # Maximum wait time in seconds
    waited=0
    wait_interval=2
    
    while [ "$waited" -lt "$max_wait" ]; do
        if check_and_update_symlink; then
            echo "Upgrade symlink check passed, proceeding with restart..."
            break
        fi
        echo "Waiting for upgrade symlink to be updated... (${waited}s/${max_wait}s)"
        sleep "$wait_interval"
        waited=$((waited + wait_interval))
    done
    
    if [ "$waited" -ge "$max_wait" ]; then
        echo "Warning: Waited ${max_wait}s for upgrade symlink update, attempting final check..."
        # Final attempt to update symlink
        check_and_update_symlink || echo "Final symlink update attempt failed, proceeding anyway..."
    fi
    
    # For any other exit code (including 0 after successful upgrade), restart
    echo "Waiting 2 seconds before restarting cosmovisor..."
    sleep 2
done

