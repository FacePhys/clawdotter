package vm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/facephys/vm-orchestrator/internal/config"
	log "github.com/sirupsen/logrus"
)

// StorageManager handles rootfs provisioning and per-user data volumes.
type StorageManager struct {
	cfg *config.Config
}

// NewStorageManager creates a new storage manager.
func NewStorageManager(cfg *config.Config) *StorageManager {
	return &StorageManager{cfg: cfg}
}

// ProvisionRootfs creates a copy-on-write snapshot of the base rootfs for a user.
// Returns the path to the user's rootfs image.
func (sm *StorageManager) ProvisionRootfs(userID string) (string, error) {
	userDir := filepath.Join(sm.cfg.VMDataDir, userID)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create user VM dir: %w", err)
	}

	rootfsPath := filepath.Join(userDir, "rootfs.ext4")

	// Check if rootfs already exists (VM restart case)
	if _, err := os.Stat(rootfsPath); err == nil {
		log.WithField("user_id", userID).Info("Using existing rootfs")
		return rootfsPath, nil
	}

	// Try CoW copy first (for btrfs/xfs/ext4 with reflink support)
	cmd := exec.Command("cp", "--reflink=auto", sm.cfg.BaseRootfsPath, rootfsPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		// Fallback to regular copy
		log.WithError(err).Warnf("reflink copy failed, using regular copy: %s", string(out))
		cmd = exec.Command("cp", sm.cfg.BaseRootfsPath, rootfsPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to copy rootfs: %s: %w", string(out), err)
		}
	}

	log.WithFields(log.Fields{
		"user_id": userID,
		"path":    rootfsPath,
	}).Info("Rootfs provisioned")

	return rootfsPath, nil
}

// ProvisionDataVolume creates a persistent data volume for a user's OpenClaw data.
// Returns the path to the data volume ext4 image.
func (sm *StorageManager) ProvisionDataVolume(userID string, sizeMB int) (string, error) {
	userDir := filepath.Join(sm.cfg.VMDataDir, userID)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create user VM dir: %w", err)
	}

	dataPath := filepath.Join(userDir, "data.ext4")

	// Skip if already exists
	if _, err := os.Stat(dataPath); err == nil {
		log.WithField("user_id", userID).Info("Using existing data volume")
		return dataPath, nil
	}

	// Create sparse file
	cmd := exec.Command("truncate", "-s", fmt.Sprintf("%dM", sizeMB), dataPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to create data volume: %s: %w", string(out), err)
	}

	// Format as ext4
	cmd = exec.Command("mkfs.ext4", "-q", "-F", dataPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		os.Remove(dataPath)
		return "", fmt.Errorf("failed to format data volume: %s: %w", string(out), err)
	}

	log.WithFields(log.Fields{
		"user_id": userID,
		"path":    dataPath,
		"size_mb": sizeMB,
	}).Info("Data volume provisioned")

	return dataPath, nil
}

// CleanupStorage removes all storage for a user (rootfs + data).
func (sm *StorageManager) CleanupStorage(userID string) error {
	userDir := filepath.Join(sm.cfg.VMDataDir, userID)
	if err := os.RemoveAll(userDir); err != nil {
		return fmt.Errorf("failed to cleanup storage for %s: %w", userID, err)
	}
	log.WithField("user_id", userID).Info("Storage cleaned up")
	return nil
}

// GetSocketPath returns the Firecracker API socket path for a user's VM.
func (sm *StorageManager) GetSocketPath(userID string) string {
	return filepath.Join(sm.cfg.SocketDir, fmt.Sprintf("%s.sock", userID))
}
