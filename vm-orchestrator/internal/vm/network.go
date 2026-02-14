package vm

import (
	"encoding/binary"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"

	"github.com/facephys/vm-orchestrator/internal/config"
	log "github.com/sirupsen/logrus"
)

// NetworkManager handles TAP device creation, IP allocation, and iptables rules.
type NetworkManager struct {
	cfg       *config.Config
	mu        sync.Mutex
	allocated map[string]string // ip -> userID
	nextIdx   int               // next TAP index
	subnetIP  net.IP
	subnetLen int
	maxHosts  int
}

// NewNetworkManager initializes the network manager and parses the VM subnet.
func NewNetworkManager(cfg *config.Config) (*NetworkManager, error) {
	ip, ipNet, err := net.ParseCIDR(cfg.VMSubnet)
	if err != nil {
		return nil, fmt.Errorf("invalid VM_SUBNET %q: %w", cfg.VMSubnet, err)
	}

	ones, bits := ipNet.Mask.Size()
	maxHosts := (1 << (bits - ones)) - 2 // Exclude network & broadcast

	return &NetworkManager{
		cfg:       cfg,
		allocated: make(map[string]string),
		nextIdx:   1,
		subnetIP:  ip.Mask(ipNet.Mask),
		subnetLen: ones,
		maxHosts:  maxHosts,
	}, nil
}

// AllocateIP assigns the next available IP from the pool.
// It skips .1 (reserved for gateway).
func (nm *NetworkManager) AllocateIP(userID string) (string, error) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	// Start from .2 (gateway is .1)
	for offset := 2; offset <= nm.maxHosts; offset++ {
		ip := nm.offsetToIP(offset)
		if _, used := nm.allocated[ip]; !used {
			nm.allocated[ip] = userID
			return ip, nil
		}
	}
	return "", fmt.Errorf("no available IPs in subnet %s", nm.cfg.VMSubnet)
}

// ReleaseIP frees an IP back to the pool.
func (nm *NetworkManager) ReleaseIP(ip string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()
	delete(nm.allocated, ip)
}

// offsetToIP converts a host offset to an IP string.
func (nm *NetworkManager) offsetToIP(offset int) string {
	ipInt := binary.BigEndian.Uint32(nm.subnetIP.To4())
	ipInt += uint32(offset)
	result := make(net.IP, 4)
	binary.BigEndian.PutUint32(result, ipInt)
	return result.String()
}

// CreateTapDevice creates a TAP device and attaches it to the bridge.
func (nm *NetworkManager) CreateTapDevice(vmIP string) (string, error) {
	nm.mu.Lock()
	idx := nm.nextIdx
	nm.nextIdx++
	nm.mu.Unlock()

	tapName := fmt.Sprintf("fc_tap%d", idx)

	commands := [][]string{
		{"ip", "tuntap", "add", "dev", tapName, "mode", "tap"},
		{"ip", "link", "set", tapName, "master", nm.cfg.BridgeName},
		{"ip", "link", "set", tapName, "up"},
	}

	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			// Cleanup on failure
			nm.DestroyTapDevice(tapName)
			return "", fmt.Errorf("failed to run %q: %s: %w", strings.Join(args, " "), string(out), err)
		}
	}

	log.WithFields(log.Fields{
		"tap":   tapName,
		"vm_ip": vmIP,
	}).Info("TAP device created and attached to bridge")

	return tapName, nil
}

// DestroyTapDevice removes a TAP device.
func (nm *NetworkManager) DestroyTapDevice(tapName string) {
	cmd := exec.Command("ip", "link", "del", tapName)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.WithError(err).WithField("tap", tapName).Warnf("Failed to destroy TAP: %s", string(out))
	}
}

// SetupVMIsolation adds iptables rules to prevent inter-VM traffic.
func (nm *NetworkManager) SetupVMIsolation(vmIP string) error {
	// Allow VM → gateway (for NAT), block VM → other VMs
	rules := [][]string{
		// Allow established connections
		{"-A", "FORWARD", "-s", vmIP, "-d", nm.cfg.GatewayIP, "-j", "ACCEPT"},
		// Allow VM to reach external (via NAT)
		{"-A", "FORWARD", "-s", vmIP, "!", "-d", nm.cfg.VMSubnet, "-j", "ACCEPT"},
		// Allow return traffic to VM
		{"-A", "FORWARD", "-d", vmIP, "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT"},
		// Allow bridge (host) to reach VM (for webhook forwarding)
		{"-A", "FORWARD", "-s", nm.cfg.GatewayIP, "-d", vmIP, "-j", "ACCEPT"},
		// Drop all other traffic to/from this VM within the subnet
		{"-A", "FORWARD", "-s", vmIP, "-d", nm.cfg.VMSubnet, "-j", "DROP"},
	}

	for _, args := range rules {
		cmd := exec.Command("iptables", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("iptables rule %v failed: %s: %w", args, string(out), err)
		}
	}

	log.WithField("vm_ip", vmIP).Info("VM isolation iptables rules applied")
	return nil
}

// CleanupVMIsolation removes iptables rules for a VM.
func (nm *NetworkManager) CleanupVMIsolation(vmIP string) {
	// Same rules but with -D (delete) instead of -A (append)
	rules := [][]string{
		{"-D", "FORWARD", "-s", vmIP, "-d", nm.cfg.GatewayIP, "-j", "ACCEPT"},
		{"-D", "FORWARD", "-s", vmIP, "!", "-d", nm.cfg.VMSubnet, "-j", "ACCEPT"},
		{"-D", "FORWARD", "-d", vmIP, "-m", "state", "--state", "ESTABLISHED,RELATED", "-j", "ACCEPT"},
		{"-D", "FORWARD", "-s", nm.cfg.GatewayIP, "-d", vmIP, "-j", "ACCEPT"},
		{"-D", "FORWARD", "-s", vmIP, "-d", nm.cfg.VMSubnet, "-j", "DROP"},
	}

	for _, args := range rules {
		cmd := exec.Command("iptables", args...)
		_ = cmd.Run() // Best-effort cleanup
	}
}
