package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"
)

func main() {
	// Define flags
	refDirPath := flag.String("refDir", "", "Path to the reference directory")
	targetDirPath := flag.String("targetDir", "", "Path to the target directory")
	parallelism := flag.Int("parallelism", runtime.NumCPU()/2, "Number of parallel workers")
	exactPathMatch := flag.Bool("exactPathMatch", true, "Exact path match flag")
	deleteFiles := flag.Bool("deleteFiles", false, "Delete files flag")

	// Define YAML input flags
	refYamlPath := flag.String("refYaml", "", "Path to reference directory YAML file")
	targetYamlPath := flag.String("targetYaml", "", "Path to target directory YAML file")

	flag.Parse()

	// Read or compute directory info for reference directory
	var refDirInfo *DirectoryInfo
	var err error

	if *refYamlPath != "" {
		refDirInfo, err = readDirectoryInfoFromYAML(*refYamlPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading reference YAML: %v\n", err)
			os.Exit(1)
		}
	} else if *refDirPath != "" {
		refDirInfo, err = WalkDirectory(*refDirPath, *parallelism, *targetDirPath == "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error walking reference directory: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Reference directory path or YAML file must be provided")
		os.Exit(1)
	}

	// If no target directory is given, output the reference directory info as YAML
	if *targetDirPath == "" && *targetYamlPath == "" {

		if *refDirPath != "" {
			// deletion candidate:
			// if we always stream output to stdout, we can remove this block
			// err := writeDirectoryInfoToYAML(refDirInfo, os.Stdout)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing reference directory info to YAML: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("Validating reference directory against yaml...")
			refFileMap := GetFileMapFromDirectoryInfo(refDirInfo, *exactPathMatch)

			// validate reference directory against the yaml
			currentRefDirInfo, err := WalkDirectory(refDirInfo.BaseDir, *parallelism, false)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error walking reference directory: %v\n", err)
				os.Exit(1)
			}
			for _, file := range currentRefDirInfo.Files {
				yamlEntry, ok := refFileMap[file.Hash]
				if !ok {
					fmt.Fprintf(os.Stderr, "File %s not found in reference directory\n", file.Path)
					os.Exit(1)
				} else {
					fmt.Printf("File %s found in reference directory: %v\n", file.Path, yamlEntry)
				}
			}
		}

		return
	}

	// Read or compute directory info for target directory
	var targetDirInfo *DirectoryInfo

	if *targetYamlPath != "" {
		targetDirInfo, err = readDirectoryInfoFromYAML(*targetYamlPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading target YAML: %v\n", err)
			os.Exit(1)
		}
	} else if *targetDirPath != "" {
		targetDirInfo, err = WalkDirectory(*targetDirPath, *parallelism, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error walking target directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Compare files
	duplicates := CompareFiles(refDirInfo, targetDirInfo, *exactPathMatch)

	// Handle deletion flag
	if *deleteFiles {
		fmt.Printf("A total of %d duplicate files found.\n", len(duplicates))
		fmt.Print("Are you sure you want to delete the files? Type 'yes' to confirm: ")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "yes" {
			err = DeleteFiles(duplicates)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error deleting files: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("File deletion aborted.")
			printDeletionPlan(duplicates, refDirInfo, targetDirInfo)
		}
	} else {
		printDeletionPlan(duplicates, refDirInfo, targetDirInfo)
	}
}

func printDeletionPlan(duplicates []FileInfo, refDir *DirectoryInfo, targetDir *DirectoryInfo) {
	refFileMap := make(map[string]string)
	for _, file := range refDir.Files {
		refFileMap[file.Hash] = file.Path
	}

	for _, file := range duplicates {
		refPath := refFileMap[file.Hash]
		fmt.Printf("rm \"%s\"  # duplicated at: %s\n", file.Path, refPath)
	}
}

func readDirectoryInfoFromYAML(path string) (*DirectoryInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var dirInfo DirectoryInfo
	err = yaml.Unmarshal(data, &dirInfo)
	if err != nil {
		return nil, err
	}

	return &dirInfo, nil
}

func writeDirectoryInfoToYAML(dirInfo *DirectoryInfo, writer *os.File) error {
	data, err := yaml.Marshal(dirInfo)
	if err != nil {
		return err
	}

	_, err = writer.Write(data)
	return err
}
