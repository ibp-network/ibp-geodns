package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"ibp-geodns/config"
	l "ibp-geodns/logging"
)

var (
	autoUpdateTimer *time.Ticker
)

const (
	officialCacheFile = "official.cache.json"
	localCacheFile    = "local.cache.json"
	statsCacheFile    = "stats.cache.json"
)

// LoadCache loads data from a cache file into the provided data structure.
func LoadCache(filePath string, data interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			l.Log(l.Warn, "Cache file not found: %s", filePath)
			return nil
		}
		l.Log(l.Error, "Failed to open cache file: %v", err)
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(data); err != nil {
		l.Log(l.Error, "Failed to decode cache file: %v", err)
		return err
	}

	l.Log(l.Info, "Cache loaded successfully from %s", filePath)
	return nil
}

// SaveCache saves the given data to a cache file.
func SaveCache(filePath string, data interface{}) error {
	file, err := os.Create(filePath)
	if err != nil {
		l.Log(l.Error, "Failed to create cache file: %v", err)
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		l.Log(l.Error, "Failed to encode data to cache file: %v", err)
		return err
	}

	// l.Log(l.Info, "Cache saved successfully to %s", filePath)
	return nil
}

// LoadAllCaches loads Official, Local, and Stats caches into the corresponding data structures.
func LoadAllCaches() {
	c := config.GetConfig()
	workDir := c.System.WorkDir

	officialFile := filepath.Join(workDir, "tmp", officialCacheFile)
	localFile := filepath.Join(workDir, "tmp", localCacheFile)
	statsFile := filepath.Join(workDir, "tmp", statsCacheFile)

	Official.Mu.Lock()
	defer Official.Mu.Unlock()
	if err := LoadCache(officialFile, &Official); err != nil {
		l.Log(l.Error, "Failed to load Official results cache: %v", err)
	}

	Local.Mu.Lock()
	defer Local.Mu.Unlock()
	if err := LoadCache(localFile, &Local); err != nil {
		l.Log(l.Error, "Failed to load Local results cache: %v", err)
	}

	Stats.Mu.Lock()
	defer Stats.Mu.Unlock()
	if err := LoadCache(statsFile, &Stats.Data); err != nil {
		l.Log(l.Error, "Failed to load Stats cache: %v", err)
	}
}

// SaveAllCaches saves Official, Local, and Stats caches.
func SaveAllCaches() {
	c := config.GetConfig()
	workDir := c.System.WorkDir

	officialFile := filepath.Join(workDir, "tmp", officialCacheFile)
	localFile := filepath.Join(workDir, "tmp", localCacheFile)
	statsFile := filepath.Join(workDir, "tmp", statsCacheFile)

	Official.Mu.Lock()
	defer Official.Mu.Unlock()
	if err := SaveCache(officialFile, &Official); err != nil {
		l.Log(l.Error, "Failed to save Official results cache: %v", err)
	}

	Local.Mu.Lock()
	defer Local.Mu.Unlock()
	if err := SaveCache(localFile, &Local); err != nil {
		l.Log(l.Error, "Failed to save Local results cache: %v", err)
	}

	Stats.Mu.Lock()
	defer Stats.Mu.Unlock()
	if err := SaveCache(statsFile, &Stats.Data); err != nil {
		l.Log(l.Error, "Failed to save Stats cache: %v", err)
	}
}

// startAutoUpdate initializes a periodic timer to save caches automatically.
func startAutoUpdate() {
	c := config.GetConfig()
	cacheSaveInterval := c.System.CacheSaveTime

	autoUpdateTimer = time.NewTicker(cacheSaveInterval * time.Second)

	go func() {
		for range autoUpdateTimer.C {
			// l.Log(l.Debug, "Auto-saving caches...")
			SaveAllCaches()
		}
	}()
}
