package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type FileInfo struct {
	Path string
	Hash string
}

type DirectoryInfo struct {
	BaseDir string
	Files   []FileInfo
}

func (f *FileInfo) CalculateHash() error {
	file, err := os.Open(f.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}
	f.Hash = fmt.Sprintf("%x", hasher.Sum(nil))
	return nil
}

func WalkDirectory(root string, parallelism int) (*DirectoryInfo, error) {
	var files []FileInfo
	fileChan := make(chan FileInfo)
	errChan := make(chan error, 1)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Start worker goroutines
	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fileInfo := range fileChan {
				if err := fileInfo.CalculateHash(); err != nil {
					select {
					case errChan <- err:
					default:
					}
					return
				}
				mu.Lock()
				files = append(files, fileInfo)
				mu.Unlock()
			}
		}()
	}

	// Walk the directory and send files to be processed
	go func() {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				fileChan <- FileInfo{Path: path}
			}
			return nil
		})
		close(fileChan)
		if err != nil {
			select {
			case errChan <- err:
			default:
			}
		}
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return &DirectoryInfo{BaseDir: root, Files: files}, nil
}

// CompareFiles compares files from two directories based on hash and relative path
// If exactPathMatch is true, it requires files to have the exact same relative path
func CompareFiles(refDir *DirectoryInfo, targetDir *DirectoryInfo, exactPathMatch bool) []FileInfo {
	refFileMap := make(map[string]map[string]bool) // map[hash]map[relpath]bool
	for _, file := range refDir.Files {
		hash := file.Hash
		relPath, _ := filepath.Rel(refDir.BaseDir, file.Path)

		if _, exists := refFileMap[hash]; !exists {
			refFileMap[hash] = make(map[string]bool)
		}

		if exactPathMatch {
			refFileMap[hash][relPath] = true
		} else {
			fileName := filepath.Base(file.Path)
			refFileMap[hash][fileName] = true
		}
	}

	var duplicates []FileInfo
	for _, file := range targetDir.Files {
		hash := file.Hash
		relPath, _ := filepath.Rel(targetDir.BaseDir, file.Path)

		if paths, exists := refFileMap[hash]; exists {
			if exactPathMatch {
				if _, pathExists := paths[relPath]; pathExists {
					duplicates = append(duplicates, file)
				}
			} else {
				fileName := filepath.Base(file.Path)
				if _, nameExists := paths[fileName]; nameExists {
					duplicates = append(duplicates, file)
				}
			}
		}
	}

	return duplicates
}

// DeleteFiles deletes the given files
func DeleteFiles(files []FileInfo) error {
	for _, file := range files {
		if err := os.Remove(file.Path); err != nil {
			return err
		}
	}
	return nil
}
