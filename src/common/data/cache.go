package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

// We track which caches the system actually wants to use.
var (
	allowLocalOfficial bool // if true, load/save official + local caches
	allowStats         bool // if true, load/save stats cache
	muCacheOptions     sync.Mutex
)

// Our cache file names
const (
	officialCacheFile = "official.cache.json"
	localCacheFile    = "local.cache.json"
	// statsCacheFile  = "stats.cache.json" -- Removed usage, no longer used
)

// SetCacheOptions is called from data.Init() to indicate whether
// we want to handle local/official caches, stats caches, or both.
func SetCacheOptions(localOfficial, stats bool) {
	muCacheOptions.Lock()
	defer muCacheOptions.Unlock()
	allowLocalOfficial = localOfficial
	allowStats = stats
	log.Log(log.Debug,
		"[cache.SetCacheOptions] localOfficial=%v, stats=%v",
		localOfficial, stats)
}

// LoadCache loads data from a cache file into the provided data structure.
func LoadCache(filePath string, out interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Log(log.Warn, "Cache file not found: %s", filePath)
			return nil
		}
		log.Log(log.Error, "Failed to open cache file '%s': %v", filePath, err)
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(out); err != nil {
		log.Log(log.Error, "Failed to decode cache file '%s': %v", filePath, err)
		return err
	}

	log.Log(log.Info, "Cache loaded successfully from %s", filePath)
	return nil
}

// SaveCache saves the given data to a cache file.
func SaveCache(filePath string, data interface{}) error {
	// First, log at DEBUG so we see exactly which file we’re about to write.
	log.Log(log.Debug, "[SaveCache] Attempting to create or overwrite cache file: %s", filePath)

	// Ensure the directory exists before creating the file.
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Log(log.Error, "Failed to create directory '%s': %v", dir, err)
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		log.Log(log.Error, "Failed to create cache file '%s': %v", filePath, err)
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		log.Log(log.Error, "Failed to encode data to cache file '%s': %v", filePath, err)
		return err
	}

	log.Log(log.Info, "Cache saved successfully to %s", filePath)
	return nil
}

// LoadAllCaches selectively loads Official, Local caches depending on allowLocalOfficial.
// Stats caching is removed, so we do not load stats data from disk.
func LoadAllCaches() {
	muCacheOptions.Lock()
	useLocal := allowLocalOfficial
	muCacheOptions.Unlock()

	c := cfg.GetConfig()
	workDir := c.Local.System.WorkDir

	officialFile := filepath.Join(workDir, "tmp", officialCacheFile)
	localFile := filepath.Join(workDir, "tmp", localCacheFile)
	// statsFile := filepath.Join(workDir, "tmp", statsCacheFile)

	if useLocal {
		log.Log(log.Debug, "[LoadAllCaches] Loading official cache from %s", officialFile)
		Official.Mu.Lock()
		if err := LoadCache(officialFile, &Official); err != nil {
			log.Log(log.Error, "[LoadAllCaches] Official load error: %v", err)
		}
		Official.Mu.Unlock()

		log.Log(log.Debug, "[LoadAllCaches] Loading local cache from %s", localFile)
		Local.Mu.Lock()
		if err := LoadCache(localFile, &Local); err != nil {
			log.Log(log.Error, "[LoadAllCaches] Local load error: %v", err)
		}
		Local.Mu.Unlock()
	}

	// We no longer load or save stats from disk, so ignore the old logic here.
}

// SaveAllCaches saves the Official and Local caches if enabled.
// We do not save usage stats to disk anymore.
func SaveAllCaches() {
	log.Log(log.Debug, "[SaveAllCaches] Entry: Attempting to save caches...")

	muCacheOptions.Lock()
	useLocal := allowLocalOfficial
	// useStats := allowStats  (no effect now)
	muCacheOptions.Unlock()

	c := cfg.GetConfig()
	workDir := c.Local.System.WorkDir

	officialFile := filepath.Join(workDir, "tmp", officialCacheFile)
	localFile := filepath.Join(workDir, "tmp", localCacheFile)
	// statsFile := filepath.Join(workDir, "tmp", statsCacheFile)

	if useLocal {
		Official.Mu.Lock()
		log.Log(log.Debug,
			"[SaveAllCaches] official: %d siteResults, %d domainResults, %d endpointResults",
			len(Official.SiteResults),
			len(Official.DomainResults),
			len(Official.EndpointResults))
		err := SaveCache(officialFile, &Official)
		Official.Mu.Unlock()
		if err != nil {
			log.Log(log.Error, "[SaveAllCaches] Official save error: %v", err)
		}

		Local.Mu.Lock()
		log.Log(log.Debug,
			"[SaveAllCaches] local: %d siteResults, %d domainResults, %d endpointResults",
			len(Local.SiteResults),
			len(Local.DomainResults),
			len(Local.EndpointResults))
		err = SaveCache(localFile, &Local)
		Local.Mu.Unlock()
		if err != nil {
			log.Log(log.Error, "[SaveAllCaches] Local save error: %v", err)
		}
	}

	log.Log(log.Debug, "[SaveAllCaches] Exit: Done saving caches.")
}
