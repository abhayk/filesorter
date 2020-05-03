package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/karrick/godirwalk"
)

type processedCount struct {
	visitedDirectories int
	copiedFiles        int
	skippedFiles       int
	erroredFiles       int
	totalBytesCopied   int64
}

func main() {

	sourcePath := flag.String("source", "", "The source directory path,")
	destPathBase := flag.String("destination", "", "The destination to which the files should be copied and sorted.")
	fileTypeFilter := flag.String("types", "", `Optional. Provide the list of file types that should be included from
	the source directory separated by a ':'. For eg: jpg:jpeg:mp4`)
	flag.Parse()

	// check for mandatory arguments
	if strings.Compare(*sourcePath, "") == 0 || strings.Compare(*destPathBase, "") == 0 {
		fmt.Println("Usage: filesorter <source path> <destination path> [file types]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if !isPathValid(*sourcePath) || !isPathValid(*destPathBase) {
		os.Exit(1)
	}

	filterTypes := make(map[string]struct{})

	if strings.Compare(*fileTypeFilter, "") != 0 {
		var empty struct{}
		for _, v := range strings.Split(*fileTypeFilter, ":") {
			filterTypes[v] = empty
		}
	}

	var counts processedCount

	godirwalk.Walk(*sourcePath, &godirwalk.Options{
		Callback: func(path string, dirent *godirwalk.Dirent) error {
			visitErr := visitFile(path, dirent, *destPathBase, filterTypes, &counts)
			if visitErr != nil {
				counts.erroredFiles++
			}
			return visitErr
		},
		PostChildrenCallback: func(path string, dirent *godirwalk.Dirent) error {
			return postVisitDir(path, dirent, &counts)
		},
		ErrorCallback: func(string, error) godirwalk.ErrorAction {
			// try processing all files even if one of the files errored.
			return godirwalk.SkipNode
		},
	})

	printReport(&counts)
}

func visitFile(path string, dirent *godirwalk.Dirent, destPathBase string, filterTypes map[string]struct{}, counts *processedCount) error {

	// walk returns directories also. skip those
	if dirent.IsDir() {
		return nil
	}

	sourceFileStat, err := os.Stat(path)
	if err != nil {
		fmt.Printf("An error occurred while trying to stat the source path %s", path)
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("The file %s is not a regular file", path)
	}

	// if file type filter were passed apply those
	if len(filterTypes) > 0 {
		if _, ok := filterTypes[filepath.Ext(path)[1:]]; !ok {
			counts.skippedFiles++
			return nil
		}
	}

	destFilePath := getDestFilePath(destPathBase, sourceFileStat)

	destFileStat, err := os.Stat(destFilePath)
	if err != nil {
		// stat returns an error if the file does not exist.
		// we can ignore that but if the error is of some other type then skip processing this file
		if !os.IsNotExist(err) {
			fmt.Printf("An error occurred while trying to stat the file %s", destFilePath)
			return err
		}
	} else {
		// we assume the file in the destination is the same as the source file if their sizes match
		// this might be useful in cases where cop file fails and an empty is created at the destination
		if sourceFileStat.Size() == destFileStat.Size() {
			counts.skippedFiles++
			return nil
		}
	}

	err = os.MkdirAll(filepath.Dir(destFilePath), os.ModePerm)
	if err != nil {
		fmt.Printf("An error occurred while trying to create directories for the file %s", destFilePath)
		return err
	}

	written, err := copyFile(path, destFilePath)
	if err != nil {
		fmt.Printf("An error occurred while trying to copy the file %s to %s", path, destFilePath)
		return err
	}

	// maintain the access and modified time of the file so that the correct time can be
	// used if the file again needs to be sorted and copied somewhere else
	err = os.Chtimes(destFilePath, sourceFileStat.ModTime(), sourceFileStat.ModTime())
	if err != nil {
		fmt.Printf("An error occurred while trying to set the access time of the copied file %s", destFilePath)
		return err
	}

	fmt.Printf("Copied %s --> %s\n", path, destFilePath)
	counts.copiedFiles++
	counts.totalBytesCopied += written

	return nil
}

func postVisitDir(path string, dirent *godirwalk.Dirent, counts *processedCount) error {
	counts.visitedDirectories++
	return nil
}

func getDestFilePath(destPathBase string, fileInfo os.FileInfo) string {
	modTime := fileInfo.ModTime()

	// a file with the name abc.txt which was last modified at May 2 2020 will end up with the path -
	// <destination directory>/2020/May/2/abc.txt
	return filepath.Join(destPathBase,
		strconv.Itoa(modTime.Year()),
		modTime.Month().String(),
		strconv.Itoa(modTime.Day()),
		fileInfo.Name())
}

func isPathValid(path string) bool {

	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Printf("The path %s does not exist.", path)
		return false
	}
	if !fileInfo.IsDir() {
		fmt.Printf("The path %s is not a directory.", path)
		return false
	}
	return true
}

func copyFile(source string, destination string) (int64, error) {

	sourceFile, err := os.Open(source)
	if err != nil {
		return 0, err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destination)
	if err != nil {
		return 0, err
	}
	defer destFile.Close()

	return io.Copy(destFile, sourceFile)
}

func printReport(counts *processedCount) {
	fmt.Println("Completed !")
	fmt.Printf("Copied %d files from %d directories. Skipped %d, Errored %d, Bytes copied %d\n",
		counts.copiedFiles,
		counts.visitedDirectories,
		counts.skippedFiles,
		counts.erroredFiles,
		counts.totalBytesCopied)
}
