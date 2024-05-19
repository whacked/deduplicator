package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

func WalkDirectory(root string) (*DirectoryInfo, error) {
	var files []FileInfo

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileInfo := FileInfo{Path: path}
			if err := fileInfo.CalculateHash(); err != nil {
				return err
			}
			files = append(files, fileInfo)
		}
		return nil
	})

	return &DirectoryInfo{BaseDir: root, Files: files}, err
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
