#!/bin/bash

# claude-on-slack Uninstallation Script
# Removes the claude-on-slack bot service and files

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVICE_NAME="claude-on-slack"
SERVICE_USER="claude-bot"
INSTALL_DIR="/opt/claude-on-slack"
CONFIG_DIR="/etc/claude-on-slack"
LOG_DIR="/var/log/claude-on-slack"
WORK_DIR="/var/lib/claude-on-slack"

# Print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Stop and disable service
stop_service() {
    print_status "Stopping and disabling $SERVICE_NAME service..."
    
    if systemctl is-active --quiet $SERVICE_NAME; then
        systemctl stop $SERVICE_NAME
        print_success "Stopped $SERVICE_NAME service"
    else
        print_warning "Service $SERVICE_NAME is not running"
    fi
    
    if systemctl is-enabled --quiet $SERVICE_NAME; then
        systemctl disable $SERVICE_NAME
        print_success "Disabled $SERVICE_NAME service"
    else
        print_warning "Service $SERVICE_NAME is not enabled"
    fi
}

# Remove systemd service file
remove_service_file() {
    print_status "Removing systemd service file..."
    
    if [[ -f "/etc/systemd/system/$SERVICE_NAME.service" ]]; then
        rm "/etc/systemd/system/$SERVICE_NAME.service"
        systemctl daemon-reload
        print_success "Removed systemd service file"
    else
        print_warning "Service file not found"
    fi
}

# Remove directories and files
remove_files() {
    print_status "Removing application files..."
    
    # Remove installation directory
    if [[ -d "$INSTALL_DIR" ]]; then
        rm -rf "$INSTALL_DIR"
        print_success "Removed $INSTALL_DIR"
    fi
    
    # Ask about configuration files
    echo
    read -p "Remove configuration files in $CONFIG_DIR? [y/N]: " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [[ -d "$CONFIG_DIR" ]]; then
            rm -rf "$CONFIG_DIR"
            print_success "Removed $CONFIG_DIR"
        fi
    else
        print_warning "Kept configuration files in $CONFIG_DIR"
    fi
    
    # Ask about logs
    echo
    read -p "Remove log files in $LOG_DIR? [y/N]: " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [[ -d "$LOG_DIR" ]]; then
            rm -rf "$LOG_DIR"
            print_success "Removed $LOG_DIR"
        fi
    else
        print_warning "Kept log files in $LOG_DIR"
    fi
    
    # Ask about workspace
    echo
    read -p "Remove workspace directory in $WORK_DIR? [y/N]: " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [[ -d "$WORK_DIR" ]]; then
            rm -rf "$WORK_DIR"
            print_success "Removed $WORK_DIR"
        fi
    else
        print_warning "Kept workspace files in $WORK_DIR"
    fi
}

# Remove service user
remove_user() {
    echo
    read -p "Remove service user '$SERVICE_USER'? [y/N]: " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if id "$SERVICE_USER" &>/dev/null; then
            userdel "$SERVICE_USER"
            print_success "Removed user '$SERVICE_USER'"
        else
            print_warning "User '$SERVICE_USER' not found"
        fi
    else
        print_warning "Kept service user '$SERVICE_USER'"
    fi
}

# Main uninstallation function
main() {
    print_status "Starting claude-on-slack uninstallation..."
    echo
    
    print_warning "This will remove the claude-on-slack service and optionally its files."
    read -p "Are you sure you want to continue? [y/N]: " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_status "Uninstallation cancelled."
        exit 0
    fi
    
    check_root
    stop_service
    remove_service_file
    remove_files
    remove_user
    
    echo
    print_success "claude-on-slack uninstallation completed!"
}

# Run main function
main "$@"