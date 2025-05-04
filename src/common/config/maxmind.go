package config

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func updateMaxMindDatabases() error {
	c := GetConfig()
	baseDir := filepath.Join(c.System.WorkDir, c.System.GeoliteDBPath)

	var cityPattern = regexp.MustCompile(`^GeoLite2-City_(\d{8})\.tar\.gz$`)
	var countryPattern = regexp.MustCompile(`^GeoLite2-Country_(\d{8})\.tar\.gz$`)
	var asnPattern = regexp.MustCompile(`^GeoLite2-ASN_(\d{8})\.tar\.gz$`)

	cityArchive, err := findNewestArchive(baseDir, cityPattern)
	if err != nil {
		return fmt.Errorf("failed to find city archive: %w", err)
	}
	countryArchive, err := findNewestArchive(baseDir, countryPattern)
	if err != nil {
		return fmt.Errorf("failed to find country archive: %w", err)
	}
	asnArchive, err := findNewestArchive(baseDir, asnPattern)
	if err != nil {
		return fmt.Errorf("failed to find ASN archive: %w", err)
	}

	if cityArchive != "" {
		if err := extractIfNeededAndLinkFile(baseDir, cityArchive, "City.mmdb"); err != nil {
			return fmt.Errorf("city extraction error: %w", err)
		}
	}
	if countryArchive != "" {
		if err := extractIfNeededAndLinkFile(baseDir, countryArchive, "Country.mmdb"); err != nil {
			return fmt.Errorf("country extraction error: %w", err)
		}
	}

	if asnArchive != "" {
		if err := extractIfNeededAndLinkFile(baseDir, asnArchive, "Asn.mmdb"); err != nil {
			return fmt.Errorf("asn extraction error: %w", err)
		}
	}

	return nil
}

func findNewestArchive(baseDir string, pattern *regexp.Regexp) (string, error) {
	dirEntries, err := os.ReadDir(baseDir)
	if err != nil {
		return "", err
	}

	var candidates []string
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if pattern.MatchString(name) {
			candidates = append(candidates, name)
		}
	}
	if len(candidates) == 0 {
		return "", nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		dateI := extractDate(candidates[i], pattern)
		dateJ := extractDate(candidates[j], pattern)
		return dateI > dateJ
	})

	return candidates[0], nil
}

func extractDate(filename string, pattern *regexp.Regexp) string {
	matches := pattern.FindStringSubmatch(filename)
	if len(matches) == 2 {
		return matches[1]
	}
	return "00000000"
}

func extractIfNeededAndLinkFile(baseDir, archiveName, symlinkFile string) error {
	outFolder := strings.TrimSuffix(archiveName, ".tar.gz")
	extractedPath := filepath.Join(baseDir, outFolder)

	if _, err := os.Stat(extractedPath); err != nil {
		archivePath := filepath.Join(baseDir, archiveName)
		if err := extractTarGz(archivePath, baseDir); err != nil {
			return err
		}
	}

	mmdbFile, err := findMMDBFile(extractedPath)
	if err != nil {
		return fmt.Errorf("failed to locate mmdb in %s: %w", extractedPath, err)
	}

	targetFile := mmdbFile

	return updateFileSymlink(baseDir, symlinkFile, targetFile)
}

func findMMDBFile(extractedDir string) (string, error) {
	var mmdbPath string
	err := filepath.Walk(extractedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".mmdb") {
			mmdbPath = path
			return io.EOF
		}
		return nil
	})
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		return "", err
	}
	if mmdbPath == "" {
		return "", fmt.Errorf("no .mmdb file found in %s", extractedDir)
	}
	return mmdbPath, nil
}

func updateFileSymlink(baseDir, symlinkFile, targetFile string) error {
	symlinkPath := filepath.Join(baseDir, symlinkFile)

	info, err := os.Stat(targetFile)
	if err != nil {
		return fmt.Errorf("target file not found: %s (err: %w)", targetFile, err)
	}
	if info.IsDir() {
		return fmt.Errorf("target path is a directory, not a file: %s", targetFile)
	}

	if err := os.RemoveAll(symlinkPath); err != nil {
		return fmt.Errorf("failed to remove old link or file at %s: %w", symlinkPath, err)
	}

	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink %s -> %s: %w", symlinkPath, targetFile, err)
	}
	return nil
}

func extractTarGz(tarGzPath, destDir string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		outPath := filepath.Join(destDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(outPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		default:
			// skip symlinks, etc. or handle as needed
		}
	}
	return nil
}
