//
// octohug
//
// copies octopress posts to hugo posts
//   converts the header
//   converts categories and tags to hugo format in header
//   if run in the octopress directory, replaces include_file with the contents
//
// http://codebrane.com/blog/2015/09/10/migrating-from-octopress-to-hugo/

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var octopressPostsDirectory string
var hugoPostDirectory string

func readFile(path string) (string, error) {
	file, fileError := os.Open(path)
	if fileError != nil {
	}
	defer file.Close()
	var buffer []byte
	fileReader := bufio.NewReaderSize(file, 10*1024)
	line, isPrefix, lineError := fileReader.ReadLine()
	for lineError == nil && !isPrefix {
		buffer = append(buffer, line...)
		buffer = append(buffer, byte('\n'))
		line, isPrefix, lineError = fileReader.ReadLine()
	}
	if isPrefix {
		fmt.Fprintln(os.Stderr, "buffer size too small")
		return "", nil
	}

	return string(buffer), nil
}

func visit(path string, fileInfo os.FileInfo, err error) error {
	if fileInfo.IsDir() {
		return nil
	}

	// Get the base filename of the post
	octopressFilename := filepath.Base(path)

	// Need to strip off the initial date and final .markdown from the post filename
	regex := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(.*).m(arkdown|d)`)
	matches := regex.FindStringSubmatch(octopressFilename)

	// Ignore non-matching filenames (i.e. do no dereference nil)
	if matches == nil {
		return nil
	}
	octopressFilenameWithoutExtension := matches[2]
	slugDateFromFile := matches[1]
	hugoFilename := hugoPostDirectory + "/" + slugDateFromFile + "-" + octopressFilenameWithoutExtension + ".md"
	fmt.Printf("%s\n%s\n", path, hugoFilename)

	// Open the octopress file
	octopressFile, octopressFileError := os.Open(path)
	// Nothing to do if we can open the source file
	if octopressFileError != nil {
		fmt.Fprintf(os.Stderr, "Error opening octopress file %s, ignoring\n", path)
		return nil
	}
	defer octopressFile.Close()

	// Create the hugo file
	hugoFile, hugoFileError := os.Create(hugoFilename)
	if hugoFileError != nil {
		fmt.Fprintf(os.Stderr, "could not create hugo file: %v\n", hugoFileError)
		return nil
	}
	defer hugoFile.Close()
	hugoFileWriter := bufio.NewWriter(hugoFile)

	// Read the octopress file line by line
	headerTagSeen := false
	octopressFileReader := bufio.NewReaderSize(octopressFile, 10*1024)
	octopressLine, isPrefix, lineError := octopressFileReader.ReadLine()
	hasDate := false
	for lineError == nil && !isPrefix {
		octopressLineAsString := string(octopressLine)
		if octopressLineAsString == "---" || octopressLineAsString == "--- " {
			if !headerTagSeen {
				hasDate = false
			} else {
				if !hasDate { // the header has no date so far
					hugoFileWriter.WriteString("date: \"" + slugDateFromFile + "\"\n")
					octoSlugDate := strings.Replace(slugDateFromFile, "-", "/", -1)
					octoFriendlySlug := octoSlugDate + "/" + octopressFilenameWithoutExtension
					hugoFileWriter.WriteString("slug: \"" + octoFriendlySlug + "\"\n")
				} else {
					hasDate = false
				}
			}
			headerTagSeen = !headerTagSeen
			octopressLineAsString = "---"
		}

		if strings.Contains(octopressLineAsString, "date: ") {
			parts := strings.Split(octopressLineAsString, " ")
			timestampRegex := regexp.MustCompile(`T\d{2}:\d{2}:\d{2}\+\d{2}:\d{2}`)
			dateStr := parts[1]
			matches := timestampRegex.FindStringSubmatch(dateStr)
			if len(matches) == 1 {
				dateStr = strings.Replace(dateStr, matches[0], "", -1)
			}
			hugoFileWriter.WriteString("date: \"" + dateStr + "\"\n")
			octoSlugDate := strings.Replace(dateStr, "-", "/", -1)
			octoFriendlySlug := octoSlugDate + "/" + octopressFilenameWithoutExtension
			hugoFileWriter.WriteString("slug: \"" + octoFriendlySlug + "\"\n")
			hasDate = true
		} else if strings.Contains(octopressLineAsString, "layout: ") {
		} else if strings.Contains(octopressLineAsString, "author: ") {
		} else if strings.Contains(octopressLineAsString, "slug: ") {
		} else if strings.Contains(octopressLineAsString, "wordpress_id: ") {
		} else if strings.Contains(octopressLineAsString, "published: ") {
			hugoFileWriter.WriteString("published = false\n")
		} else if strings.Contains(octopressLineAsString, "include_code") {
			parts := strings.Split(octopressLineAsString, " ")
			// can be:
			// {% include_code [RedViewController.m] lang:objectivec slidernav/RedViewController.m %}
			// or
			// {% include_code [RedViewController.m] slidernav/RedViewController.m %}
			codeFilePath := "source/downloads/code/" + parts[len(parts)-2]
			codeFileContent, _ := readFile(codeFilePath)
			codeFileContent = strings.Replace(codeFileContent, "<", "&lt;", -1)
			codeFileContent = strings.Replace(codeFileContent, ">", "&gt;", -1)
			hugoFileWriter.WriteString("<pre><code>\n" + codeFileContent + "</code></pre>\n")
		} else {
			hugoFileWriter.WriteString(octopressLineAsString + "\n")
		} // if octopressLineAsString == "categories:"

		hugoFileWriter.Flush()
		octopressLine, isPrefix, lineError = octopressFileReader.ReadLine()
	}
	if isPrefix {
		fmt.Fprintln(os.Stderr, "buffer size too small")
	}
	return nil
}

func init() {
	flag.StringVar(&octopressPostsDirectory, "octo", "source/_posts", "path to octopress posts directory")
	flag.StringVar(&hugoPostDirectory, "hugo", "content/post", "path to hugo post directory")
}

func main() {
	flag.Parse()

	// Check that we can trust octopressPostsDirectory
	if _, err := os.Stat(octopressPostsDirectory); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
		os.Exit(-1)
	}
	os.MkdirAll(hugoPostDirectory, 0777)
	filepath.Walk(octopressPostsDirectory, visit)
}
