package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

// We track which caches the system actually wants to use.
var (
	autoUpdateTimer    *time.Ticker
	allowLocalOfficial bool // if true, load/save official + local caches
	allowStats         bool // if true, load/save stats cache
	muCacheOptions     sync.Mutex
)

// Our cache file names
const (
	officialCacheFile = "official.cache.json"
	localCacheFile    = "local.cache.json"
	statsCacheFile    = "stats.cache.json"
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
		log.Log(log.Error, "Failed to open cache file: %v", err)
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(out); err != nil {
		log.Log(log.Error, "Failed to decode cache file: %v", err)
		return err
	}

	log.Log(log.Info, "Cache loaded successfully from %s", filePath)
	return nil
}

// SaveCache saves the given data to a cache file.
func SaveCache(filePath string, data interface{}) error {
	file, err := os.Create(filePath)
	if err != nil {
		log.Log(log.Error, "Failed to create cache file: %v", err)
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		log.Log(log.Error, "Failed to encode data to cache file: %v", err)
		return err
	}

	log.Log(log.Info, "Cache saved successfully to %s", filePath)
	return nil
}

// LoadAllCaches selectively loads Official, Local, and Stats caches
// depending on allowLocalOfficial & allowStats.
func LoadAllCaches() {
	muCacheOptions.Lock()
	useLocal := allowLocalOfficial
	useStats := allowStats
	muCacheOptions.Unlock()

	c := cfg.GetConfig()
	workDir := c.Local.System.WorkDir

	officialFile := filepath.Join(workDir, "tmp", officialCacheFile)
	localFile := filepath.Join(workDir, "tmp", localCacheFile)
	statsFile := filepath.Join(workDir, "tmp", statsCacheFile)

	// If we are using local/official caches, load them.
	if useLocal {
		Official.Mu.Lock()
		defer Official.Mu.Unlock()
		if err := LoadCache(officialFile, &Official); err != nil {
			log.Log(log.Error, "Failed to load Official results cache: %v", err)
		}

		Local.Mu.Lock()
		defer Local.Mu.Unlock()
		if err := LoadCache(localFile, &Local); err != nil {
			log.Log(log.Error, "Failed to load Local results cache: %v", err)
		}
	}

	// If we are using stats, load stats cache.
	if useStats {
		Stats.Mu.Lock()
		defer Stats.Mu.Unlock()
		if err := LoadCache(statsFile, &Stats.Data); err != nil {
			log.Log(log.Error, "Failed to load Stats cache: %v", err)
		}
	}
}

func SaveAllCaches() {
	// ADDED:
	log.Log(log.Debug, "[SaveAllCaches] Entry: Attempting to save caches...")

	muCacheOptions.Lock()
	useLocal := allowLocalOfficial
	useStats := allowStats
	muCacheOptions.Unlock()

	c := cfg.GetConfig()
	workDir := c.Local.System.WorkDir

	officialFile := filepath.Join(workDir, "tmp", officialCacheFile)
	localFile := filepath.Join(workDir, "tmp", localCacheFile)
	statsFile := filepath.Join(workDir, "tmp", statsCacheFile)

	// If we are using local/official caches
	if useLocal {
		Official.Mu.Lock()
		// ADDED:
		log.Log(log.Debug, "[SaveAllCaches] official: %d siteResults, %d domainResults, %d endpointResults",
			len(Official.SiteResults), len(Official.DomainResults), len(Official.EndpointResults))
		err := SaveCache(officialFile, &Official)
		Official.Mu.Unlock()
		if err != nil {
			log.Log(log.Error, "[SaveAllCaches] Official save error: %v", err)
		}

		Local.Mu.Lock()
		// ADDED:
		log.Log(log.Debug, "[SaveAllCaches] local: %d siteResults, %d domainResults, %d endpointResults",
			len(Local.SiteResults), len(Local.DomainResults), len(Local.EndpointResults))
		err = SaveCache(localFile, &Local)
		Local.Mu.Unlock()
		if err != nil {
			log.Log(log.Error, "[SaveAllCaches] Local save error: %v", err)
		}
	}

	// If we are using stats
	if useStats {
		Stats.Mu.Lock()
		// ADDED:
		log.Log(log.Debug, "[SaveAllCaches] stats: date entries = %d", len(Stats.Data))
		err := SaveCache(statsFile, &Stats.Data)
		Stats.Mu.Unlock()
		if err != nil {
			log.Log(log.Error, "[SaveAllCaches] Stats save error: %v", err)
		}
	}

	log.Log(log.Debug, "[SaveAllCaches] Exit: Done saving caches.")
}

// startAutoUpdate runs a ticker to auto-save caches. Only caches
// that are enabled will be saved each interval.
func startAutoUpdate() {
	c := cfg.GetConfig()
	cacheSaveInterval := c.Local.System.CacheSaveTime

	autoUpdateTimer = time.NewTicker(cacheSaveInterval * time.Second)
	go func() {
		for range autoUpdateTimer.C {
			log.Log(log.Info, "Auto-saving caches...")
			SaveAllCaches()
		}
	}()
}
