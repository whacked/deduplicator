package main

import (
	"log"
	"os"
	"path/filepath"
	"testing"
)

// Helper function to create test files based on a given structure
func createTestFiles(fileStructure []struct{ Path, Content string }) (string, error) {
	testDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		return "", err
	}
	log.Printf("Created test directory in %s\n", testDir)

	for _, file := range fileStructure {
		filePath := filepath.Join(testDir, file.Path)
		dir := filepath.Dir(filePath)
		os.MkdirAll(dir, 0755)
		if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
			return "", err
		}
	}
	return testDir, nil
}

// Create test files for comparison that requires exact relpath match
func createExactTestFiles() (string, string, error) {
	refFiles := []struct{ Path, Content string }{
		{"file1.txt", "This is file 1"},
		{"file2.txt", "This is file 2"},
		{"subdir/file3.txt", "This is file 3"},
		{"subdir/empty.txt", ""},
	}
	targetFiles := []struct{ Path, Content string }{
		{"file1.txt", "This is file 1"},
		{"file2.txt", "This is file 2"},
		{"subdir/file3.txt", "This is file 3"},
		{"projects/foo/bar/empty.txt", ""}, // Different path, same content, will not get picked as dupes by comparator
	}

	refDir, err := createTestFiles(refFiles)
	if err != nil {
		return "", "", err
	}

	targetDir, err := createTestFiles(targetFiles)
	if err != nil {
		return "", "", err
	}

	return refDir, targetDir, nil
}

// Create test files for comparison that does not require exact relpath match
func createNonExactTestFiles() (string, string, error) {
	refFiles := []struct{ Path, Content string }{
		{"file1.txt", "This is file 1"},
		{"file2.txt", "This is file 2"},
		{"subdir/file3.txt", "This is file 3"},
		{"empty.txt", ""},
		{"foobar.txt", "baz"},
	}
	targetFiles := []struct{ Path, Content string }{
		{"file1.txt", "This is file 1"},
		{"blah/file2.txt", "This is file 2"}, // different location, same content
		{"subdir/file3.txt", "This is file 3"},
		{"empty1.txt", ""},       // different name
		{"subdir/empty.txt", ""}, // different location, same content
		{"hoge.txt", "fuga"},
	}

	refDir, err := createTestFiles(refFiles)
	if err != nil {
		return "", "", err
	}

	targetDir, err := createTestFiles(targetFiles)
	if err != nil {
		return "", "", err
	}

	return refDir, targetDir, nil
}

func removeTestFiles(testDir string) {
	log.Printf("Removing test files in %s\n", testDir)
	os.RemoveAll(testDir)
}

func TestWalkDirectory(t *testing.T) {
	refFiles := []struct{ Path, Content string }{
		{"file1.txt", "This is file 1"},
		{"file2.txt", "This is file 2"},
		{"empty.txt", ""},
		{"subdir/file3.txt", "This is file 3"},
		{"subdir/empty.txt", ""},
	}
	testDir, err := createTestFiles(refFiles)
	if err != nil {
		t.Fatalf("Failed to create test files: %v", err)
	}
	defer removeTestFiles(testDir)

	dirInfo, err := WalkDirectory(testDir)
	if err != nil {
		t.Fatalf("Error walking directory: %v", err)
	}

	expected := map[string]string{
		filepath.Join(testDir, "file1.txt"):        "eedf707e950e8315f7287656d49190d08dcafc0ebd0fd68ee653cd2ce6801b01",
		filepath.Join(testDir, "file2.txt"):        "e063841728a370901b1e7b5fcf8b17406efecd04f98b6a47373a9013fb3afe5b",
		filepath.Join(testDir, "subdir/file3.txt"): "3db623ae371bcede75cbce0f1200e873822b93547867d5ad29716418c4eb8293",
		filepath.Join(testDir, "empty.txt"):        "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		filepath.Join(testDir, "subdir/empty.txt"): "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}

	for _, file := range dirInfo.Files {
		if hash, ok := expected[file.Path]; ok {
			if file.Hash != hash {
				t.Errorf("Unexpected hash for %s: got %s, want %s", file.Path, file.Hash, hash)
			}
		} else {
			t.Errorf("Unexpected file: %s", file.Path)
		}
	}
}

func TestCompareFiles(t *testing.T) {
	// Test exact comparison
	refDirExact, targetDirExact, err := createExactTestFiles()
	if err != nil {
		t.Fatalf("Failed to create exact test files: %v", err)
	}
	defer removeTestFiles(refDirExact)
	defer removeTestFiles(targetDirExact)

	refDirInfoExact, err := WalkDirectory(refDirExact)
	if err != nil {
		t.Fatalf("Error walking reference directory (exact): %v", err)
	}

	targetDirInfoExact, err := WalkDirectory(targetDirExact)
	if err != nil {
		t.Fatalf("Error walking target directory (exact): %v", err)
	}

	duplicatesExact := CompareFiles(refDirInfoExact, targetDirInfoExact, true)
	expectedExact := map[string]bool{
		filepath.Join(targetDirExact, "file1.txt"):        true,
		filepath.Join(targetDirExact, "file2.txt"):        true,
		filepath.Join(targetDirExact, "subdir/file3.txt"): true,
	}

	if len(duplicatesExact) != len(expectedExact) {
		t.Errorf("Unexpected number of duplicates (exact match): got %d, want %d", len(duplicatesExact), len(expectedExact))
	}

	for _, file := range duplicatesExact {
		if _, ok := expectedExact[file.Path]; !ok {
			t.Errorf("Unexpected duplicate file (exact match): %s", file.Path)
		}
		delete(expectedExact, file.Path)
	}

	if len(expectedExact) > 0 {
		t.Errorf("Some expected duplicates (exact match) were not found: %v", expectedExact)
	}

	// Test non-exact comparison
	refDirNonExact, targetDirNonExact, err := createNonExactTestFiles()
	if err != nil {
		t.Fatalf("Failed to create non-exact test files: %v", err)
	}
	defer removeTestFiles(refDirNonExact)
	defer removeTestFiles(targetDirNonExact)

	refDirInfoNonExact, err := WalkDirectory(refDirNonExact)
	if err != nil {
		t.Fatalf("Error walking reference directory (non-exact): %v", err)
	}

	targetDirInfoNonExact, err := WalkDirectory(targetDirNonExact)
	if err != nil {
		t.Fatalf("Error walking target directory (non-exact): %v", err)
	}

	duplicatesNonExact := CompareFiles(refDirInfoNonExact, targetDirInfoNonExact, false)
	expectedNonExact := map[string]bool{
		filepath.Join(targetDirNonExact, "file1.txt"):        true,
		filepath.Join(targetDirNonExact, "blah/file2.txt"):   true,
		filepath.Join(targetDirNonExact, "subdir/file3.txt"): true,
		filepath.Join(targetDirNonExact, "subdir/empty.txt"): true,
	}

	if len(duplicatesNonExact) != len(expectedNonExact) {
		t.Errorf("Unexpected number of duplicates (non-exact match): got %d, want %d", len(duplicatesNonExact), len(expectedNonExact))
		for _, file := range duplicatesNonExact {
			t.Logf("Duplicate file (non-exact match): %s", file.Path)
		}
	}

	for _, file := range duplicatesNonExact {
		if _, ok := expectedNonExact[file.Path]; !ok {
			t.Errorf("Unexpected duplicate file (non-exact match): %s", file.Path)
		}
		delete(expectedNonExact, file.Path)
	}

	if len(expectedNonExact) > 0 {
		t.Errorf("Some expected duplicates (non-exact match) were not found: %v", expectedNonExact)
	}
}
