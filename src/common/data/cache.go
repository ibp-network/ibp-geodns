package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

var (
	allowLocalOfficial bool
	allowStats         bool
	muCacheOptions     sync.Mutex
)

const (
	officialCacheFile = "official.cache.json"
	localCacheFile    = "local.cache.json"
)

func SetCacheOptions(localOfficial, stats bool) {
	muCacheOptions.Lock()
	defer muCacheOptions.Unlock()
	allowLocalOfficial = localOfficial
	allowStats = stats
	log.Log(log.Debug,
		"[cache.SetCacheOptions] localOfficial=%v, stats=%v",
		localOfficial, stats)
}

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

func SaveCache(filePath string, data interface{}) error {
	log.Log(log.Debug, "[SaveCache] Attempting to create or overwrite cache file: %s", filePath)

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

func LoadAllCaches() {
	muCacheOptions.Lock()
	useLocal := allowLocalOfficial
	muCacheOptions.Unlock()

	c := cfg.GetConfig()
	workDir := c.Local.System.WorkDir

	officialFile := filepath.Join(workDir, "tmp", officialCacheFile)
	localFile := filepath.Join(workDir, "tmp", localCacheFile)

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
}

func SaveAllCaches() {
	log.Log(log.Debug, "[SaveAllCaches] Entry: Attempting to save caches...")

	muCacheOptions.Lock()
	useLocal := allowLocalOfficial
	muCacheOptions.Unlock()

	c := cfg.GetConfig()
	workDir := c.Local.System.WorkDir

	officialFile := filepath.Join(workDir, "tmp", officialCacheFile)
	localFile := filepath.Join(workDir, "tmp", localCacheFile)

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
