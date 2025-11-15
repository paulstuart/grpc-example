#!/bin/bash
# Setup DNS resolution for Docker services on the host
# Usage: sudo ./scripts/setup-dns.sh [add|remove]

HOSTS_FILE="/etc/hosts"
MARKER_START="# BEGIN grpc-example Docker services"
MARKER_END="# END grpc-example Docker services"

add_entries() {
    # Check if entries already exist
    if grep -q "$MARKER_START" "$HOSTS_FILE"; then
        echo "DNS entries already exist in $HOSTS_FILE"
        echo "Run: sudo $0 remove"
        echo "Then: sudo $0 add"
        exit 1
    fi

    echo "Adding Docker DNS entries to $HOSTS_FILE..."

    cat >> "$HOSTS_FILE" << EOF

$MARKER_START
127.0.0.1 rsyslog grpc-rsyslog
127.0.0.1 postgres grpc-postgres
127.0.0.1 tempo grpc-tempo
127.0.0.1 grafana grpc-grafana
127.0.0.1 otel-collector grpc-otel-collector
127.0.0.1 api grpc-example-api
$MARKER_END
EOF

    echo "✅ DNS entries added successfully!"
    echo ""
    echo "You can now use these hostnames from your host machine:"
    echo "  http://grafana:3000"
    echo "  http://tempo:3200"
    echo "  https://api:11004 (note: port 11004, not 11000)"
    echo "  postgresql://postgres:5432"
}

remove_entries() {
    if ! grep -q "$MARKER_START" "$HOSTS_FILE"; then
        echo "No DNS entries found in $HOSTS_FILE"
        exit 0
    fi

    echo "Removing Docker DNS entries from $HOSTS_FILE..."

    # Create temporary file without the marked section
    sed "/$MARKER_START/,/$MARKER_END/d" "$HOSTS_FILE" > /tmp/hosts.tmp
    mv /tmp/hosts.tmp "$HOSTS_FILE"

    echo "✅ DNS entries removed successfully!"
}

show_usage() {
    echo "Usage: sudo $0 [add|remove|status]"
    echo ""
    echo "Commands:"
    echo "  add     - Add Docker service DNS entries to /etc/hosts"
    echo "  remove  - Remove Docker service DNS entries from /etc/hosts"
    echo "  status  - Show current DNS entries"
    echo ""
    echo "Note: This script requires sudo privileges"
}

show_status() {
    if grep -q "$MARKER_START" "$HOSTS_FILE"; then
        echo "✅ Docker DNS entries are installed in $HOSTS_FILE:"
        echo ""
        sed -n "/$MARKER_START/,/$MARKER_END/p" "$HOSTS_FILE"
    else
        echo "❌ Docker DNS entries are not installed"
        echo ""
        echo "Run: sudo $0 add"
    fi
}

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root"
    echo "Usage: sudo $0 [add|remove|status]"
    exit 1
fi

case "$1" in
    add)
        add_entries
        ;;
    remove)
        remove_entries
        ;;
    status)
        show_status
        ;;
    *)
        show_usage
        exit 1
        ;;
esac
