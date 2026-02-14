#!/bin/bash
# setup-network.sh — One-time host network setup for Firecracker VMs
#
# Creates a bridge interface and configures NAT for VM internet access.
# Must be run as root on the host machine.
#
# Usage: sudo ./setup-network.sh [bridge_name] [gateway_ip] [subnet]

set -euo pipefail

BRIDGE_NAME="${1:-fcbr0}"
GATEWAY_IP="${2:-10.0.1.1}"
SUBNET="${3:-10.0.1.0/24}"
HOST_IFACE="${4:-eth0}"  # The host's outgoing network interface

echo "=== Firecracker Host Network Setup ==="
echo "  Bridge:    $BRIDGE_NAME"
echo "  Gateway:   $GATEWAY_IP"
echo "  Subnet:    $SUBNET"
echo "  Host NIC:  $HOST_IFACE"
echo ""

# 1. Create Linux bridge
if ! ip link show "$BRIDGE_NAME" &>/dev/null; then
    echo "[+] Creating bridge $BRIDGE_NAME"
    ip link add name "$BRIDGE_NAME" type bridge
    ip addr add "${GATEWAY_IP}/24" dev "$BRIDGE_NAME"
    ip link set "$BRIDGE_NAME" up
else
    echo "[=] Bridge $BRIDGE_NAME already exists"
fi

# 2. Enable IP forwarding
echo "[+] Enabling IP forwarding"
sysctl -w net.ipv4.ip_forward=1
# Make persistent
if ! grep -q "net.ipv4.ip_forward=1" /etc/sysctl.conf 2>/dev/null; then
    echo "net.ipv4.ip_forward=1" >> /etc/sysctl.conf
fi

# 3. Setup NAT masquerade for VM internet access
echo "[+] Configuring NAT masquerade"
iptables -t nat -A POSTROUTING -s "$SUBNET" -o "$HOST_IFACE" -j MASQUERADE 2>/dev/null || \
    echo "[=] NAT rule may already exist"

# 4. Allow forwarded traffic from bridge
iptables -A FORWARD -i "$BRIDGE_NAME" -o "$HOST_IFACE" -j ACCEPT 2>/dev/null || true
iptables -A FORWARD -i "$HOST_IFACE" -o "$BRIDGE_NAME" -m state --state ESTABLISHED,RELATED -j ACCEPT 2>/dev/null || true

# 5. Default drop inter-VM traffic (VMs on the same bridge)
# Individual VM rules are added by the orchestrator.
# This is a catch-all safety net.
iptables -A FORWARD -i "$BRIDGE_NAME" -o "$BRIDGE_NAME" -j DROP 2>/dev/null || true

echo ""
echo "=== Network setup complete ==="
echo "Bridge $BRIDGE_NAME is ready at $GATEWAY_IP"
echo "VMs in $SUBNET can access the internet via NAT"
echo "Inter-VM traffic is blocked by default"
