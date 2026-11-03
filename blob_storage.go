package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

var validHashRegex = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

// IsValidHash validates that a content identifier is a valid 64-character SHA-256 hex string.
// This strictly prevents path traversal attacks (e.g. ../../etc/passwd).
func IsValidHash(hash string) bool {
	return validHashRegex.MatchString(hash)
}

// BlobStorage defines content-addressable storage operations for blog markdown files.
// Designed to be backed by local filesystem, peer-to-peer meshes, or object storage (S3/IPFS).
type BlobStorage interface {
	Save(content []byte) (hash string, err error)
	Get(hash string) ([]byte, error)
	Delete(hash string) error
	Exists(hash string) bool
	AddPeer(peerURL string)
	GetPeers() []string
}

// LocalCASStorage implements content-addressable storage on the local disk.
type LocalCASStorage struct {
	baseDir string
}

func NewLocalCASStorage(baseDir string) (*LocalCASStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create blob directory: %w", err)
	}
	return &LocalCASStorage{baseDir: baseDir}, nil
}

// Save computes the SHA-256 hash of the content and persists it as <hash>.md
func (s *LocalCASStorage) Save(content []byte) (string, error) {
	hasher := sha256.New()
	hasher.Write(content)
	hash := hex.EncodeToString(hasher.Sum(nil))

	filePath := filepath.Join(s.baseDir, hash+".md")

	// Deduplication: if file already exists with identical hash, return immediately
	if _, err := os.Stat(filePath); err == nil {
		return hash, nil
	}

	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write blob file: %w", err)
	}

	return hash, nil
}

// Get retrieves the raw markdown bytes using the content hash, with strict path traversal checks
func (s *LocalCASStorage) Get(hash string) ([]byte, error) {
	if !IsValidHash(hash) {
		return nil, fmt.Errorf("security violation: invalid content hash format")
	}

	filePath := filepath.Join(s.baseDir, hash+".md")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("blob with hash %s not found: %w", hash, err)
	}
	return data, nil
}

// Delete removes a blob file
func (s *LocalCASStorage) Delete(hash string) error {
	if !IsValidHash(hash) {
		return fmt.Errorf("security violation: invalid content hash format")
	}

	filePath := filepath.Join(s.baseDir, hash+".md")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// Exists checks if a content hash already exists locally
func (s *LocalCASStorage) Exists(hash string) bool {
	if !IsValidHash(hash) {
		return false
	}
	filePath := filepath.Join(s.baseDir, hash+".md")
	_, err := os.Stat(filePath)
	return err == nil
}

func (s *LocalCASStorage) AddPeer(peerURL string) {}
func (s *LocalCASStorage) GetPeers() []string     { return nil }

// ----------------- DISTRIBUTED CAS STORAGE MESH -----------------

// DistributedCASStorage coordinates content replication and retrieval across multiple machines.
// If a requested blob is missing locally, it self-heals by fetching from peer nodes and
// cryptographically validating the SHA-256 hash before saving locally.
type DistributedCASStorage struct {
	local      *LocalCASStorage
	peers      []string
	mu         sync.RWMutex
	client     *http.Client
	cache      map[string][]byte
	cacheMu    sync.RWMutex
	cacheLimit int
}

func NewDistributedCASStorage(local *LocalCASStorage, peers []string) *DistributedCASStorage {
	// Clean and parse peers
	var cleanPeers []string
	for _, p := range peers {
		trimmed := strings.TrimRight(strings.TrimSpace(p), "/")
		if trimmed != "" {
			cleanPeers = append(cleanPeers, trimmed)
		}
	}

	// Also check PEER_NODES environment variable
	if envPeers := os.Getenv("PEER_NODES"); envPeers != "" {
		for _, p := range strings.Split(envPeers, ",") {
			trimmed := strings.TrimRight(strings.TrimSpace(p), "/")
			if trimmed != "" {
				cleanPeers = append(cleanPeers, trimmed)
			}
		}
	}

	return &DistributedCASStorage{
		local:      local,
		peers:      cleanPeers,
		cache:      make(map[string][]byte),
		cacheLimit: 256, // In-memory hot cache for up to 256 active manuscripts
		client: &http.Client{
			Timeout: 4 * time.Second,
		},
	}
}

func (d *DistributedCASStorage) AddPeer(peerURL string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	clean := strings.TrimRight(strings.TrimSpace(peerURL), "/")
	if clean == "" {
		return
	}
	for _, p := range d.peers {
		if p == clean {
			return
		}
	}
	d.peers = append(d.peers, clean)
}

func (d *DistributedCASStorage) GetPeers() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	cp := make([]string, len(d.peers))
	copy(cp, d.peers)
	return cp
}

// Save stores content locally, updates in-memory cache, and asynchronously syncs to cluster peers
func (d *DistributedCASStorage) Save(content []byte) (string, error) {
	hash, err := d.local.Save(content)
	if err != nil {
		return "", err
	}

	// Update hot cache
	d.cacheMu.Lock()
	if len(d.cache) < d.cacheLimit {
		d.cache[hash] = content
	}
	d.cacheMu.Unlock()

	// Broadcast asynchronous replication to cluster peers
	d.mu.RLock()
	peers := make([]string, len(d.peers))
	copy(peers, d.peers)
	d.mu.RUnlock()

	if len(peers) > 0 {
		go d.broadcastReplication(peers, hash, content)
	}

	return hash, nil
}

func (d *DistributedCASStorage) broadcastReplication(peers []string, hash string, content []byte) {
	for _, peer := range peers {
		syncURL := peer + "/api/v1/sync/blob"
		req, err := http.NewRequest("POST", syncURL, bytes.NewReader(content))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "text/markdown; charset=utf-8")
		req.Header.Set("X-Content-Hash", hash)

		resp, err := d.client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}
}

// Get retrieves blob from in-memory cache -> local disk -> peer mesh (with verification)
func (d *DistributedCASStorage) Get(hash string) ([]byte, error) {
	if !IsValidHash(hash) {
		return nil, fmt.Errorf("security violation: invalid content hash format")
	}

	// 1. Check in-memory hot cache
	d.cacheMu.RLock()
	if cached, ok := d.cache[hash]; ok {
		d.cacheMu.RUnlock()
		return cached, nil
	}
	d.cacheMu.RUnlock()

	// 2. Check local disk storage
	data, err := d.local.Get(hash)
	if err == nil {
		d.cacheMu.Lock()
		if len(d.cache) < d.cacheLimit {
			d.cache[hash] = data
		}
		d.cacheMu.Unlock()
		return data, nil
	}

	// 3. Fallback: Query remote peer nodes across the distributed mesh
	d.mu.RLock()
	peers := make([]string, len(d.peers))
	copy(peers, d.peers)
	d.mu.RUnlock()

	for _, peer := range peers {
		peerURL := peer + "/blob/" + hash
		resp, err := d.client.Get(peerURL)
		if err != nil {
			LogError().Printf("Peer %s GET error: %v\n", peerURL, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			LogError().Printf("Peer %s status: %d\n", peerURL, resp.StatusCode)
			_ = resp.Body.Close()
			continue
		}

		peerData, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20)) // 20MB limit
		_ = resp.Body.Close()
		if err != nil {
			continue
		}

		// Cryptographic Integrity Verification:
		// Ensure that sha256(peerData) strictly equals the requested hash!
		hasher := sha256.New()
		hasher.Write(peerData)
		computedHash := hex.EncodeToString(hasher.Sum(nil))

		if computedHash != hash {
			LogWarn().Printf("Peer %s served tampered or mismatched blob for hash %s\n", peer, hash)
			continue
		}

		// Self-heal: Persist verified blob to local CAS disk and hot cache
		_, _ = d.local.Save(peerData)
		d.cacheMu.Lock()
		if len(d.cache) < d.cacheLimit {
			d.cache[hash] = peerData
		}
		d.cacheMu.Unlock()

		return peerData, nil
	}

	return nil, fmt.Errorf("blob with hash %s not found locally or across %d cluster peers", hash, len(peers))
}

func (d *DistributedCASStorage) Delete(hash string) error {
	d.cacheMu.Lock()
	delete(d.cache, hash)
	d.cacheMu.Unlock()
	return d.local.Delete(hash)
}

func (d *DistributedCASStorage) Exists(hash string) bool {
	if !IsValidHash(hash) {
		return false
	}
	d.cacheMu.RLock()
	if _, ok := d.cache[hash]; ok {
		d.cacheMu.RUnlock()
		return true
	}
	d.cacheMu.RUnlock()
	return d.local.Exists(hash)
}
