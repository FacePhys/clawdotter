package config

import (
	"fmt"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
)

// Config holds all orchestrator configuration.
type Config struct {
	// Server
	Port int
	Host string

	// Redis
	RedisURL string

	// Firecracker paths
	FirecrackerBin string // Path to firecracker binary
	KernelPath     string // Path to vmlinux kernel image
	BaseRootfsPath string // Path to base rootfs ext4 image

	// VM defaults
	DefaultVCPU   int
	DefaultMemMiB int

	// Networking
	VMSubnet    string // CIDR for VM IP pool, e.g. "10.0.1.0/24"
	BridgeName  string // Linux bridge name, e.g. "br0"
	GatewayIP   string // Gateway IP on the bridge, e.g. "10.0.1.1"

	// Storage
	VMDataDir   string // Directory for per-user rootfs copies & data
	SocketDir   string // Directory for Firecracker API sockets
}

func envOrDefault(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func envIntOrDefault(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:           envIntOrDefault("PORT", 8080),
		Host:           envOrDefault("HOST", "0.0.0.0"),
		RedisURL:       envOrDefault("REDIS_URL", "redis://localhost:6379"),
		FirecrackerBin: envOrDefault("FIRECRACKER_BIN", "/usr/local/bin/firecracker"),
		KernelPath:     envOrDefault("KERNEL_PATH", "/var/lib/firecracker/vmlinux"),
		BaseRootfsPath: envOrDefault("BASE_ROOTFS_PATH", "/var/lib/firecracker/rootfs/clawdbot.ext4"),
		DefaultVCPU:    envIntOrDefault("DEFAULT_VCPU", 1),
		DefaultMemMiB:  envIntOrDefault("DEFAULT_MEM_MIB", 512),
		VMSubnet:       envOrDefault("VM_SUBNET", "10.0.1.0/24"),
		BridgeName:     envOrDefault("BRIDGE_NAME", "fcbr0"),
		GatewayIP:      envOrDefault("GATEWAY_IP", "10.0.1.1"),
		VMDataDir:      envOrDefault("VM_DATA_DIR", "/var/lib/firecracker/vms"),
		SocketDir:      envOrDefault("SOCKET_DIR", "/tmp/firecracker/sockets"),
	}

	// Check Firecracker paths — warn if missing (VM creation will fail, but API can still start)
	if _, err := os.Stat(cfg.FirecrackerBin); os.IsNotExist(err) {
		log.Warnf("Firecracker binary not found: %s (VM creation will be unavailable)", cfg.FirecrackerBin)
	}
	if _, err := os.Stat(cfg.KernelPath); os.IsNotExist(err) {
		log.Warnf("Kernel image not found: %s (VM creation will be unavailable)", cfg.KernelPath)
	}
	if _, err := os.Stat(cfg.BaseRootfsPath); os.IsNotExist(err) {
		log.Warnf("Base rootfs not found: %s (VM creation will be unavailable)", cfg.BaseRootfsPath)
	}

	// Ensure directories exist
	for _, dir := range []string{cfg.VMDataDir, cfg.SocketDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return cfg, nil
}
