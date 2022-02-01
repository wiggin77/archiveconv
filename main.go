package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

const (
	AppName = "FocalBoard Archive Converter"
	Version = "0.1.0"

	ArchiveExtension = ".boardarchive"

	ArchiveVersionSource = 1
	ArchiveVersionTarget = 2

	DefaultDataDir = "./data"
)

func main() {
	var filename string
	var dataDir string
	var templateMode bool
	var showImageInfo bool
	var debug bool

	flag.StringVar(&filename, "f", "", "filename of version 1 archive")
	flag.StringVar(&dataDir, "d", DefaultDataDir, "directory for image files")
	flag.BoolVar(&showImageInfo, "i", false, "display image info")
	flag.BoolVar(&templateMode, "t", false, "use archive to create default templates")
	flag.BoolVar(&debug, "debug", false, "enable debug output")
	flag.Parse()

	if filename == "" {
		flag.Usage()
		os.Exit(-1)
	}

	_, err := os.Stat(filename)
	if err != nil {
		LogFatal(-1, "invalid archive file:", err)
	}

	archiveJSONL, err := ioutil.ReadFile(filename)
	if err != nil {
		LogFatal(-4, "error reading", filename, ":", err)
	}

	opts := ConvertOptions{
		OutputFilename: filename + ArchiveExtension,
		DataDir:        dataDir,
		TemplateMode:   templateMode,
		ShowImageInfo:  showImageInfo,
	}

	if err := Convert(archiveJSONL, opts); err != nil {
		LogFatal(-10, "error converting archive: ", err)
	}
}

func LogDebug(stuff ...interface{}) {
	_log(os.Stdout, stuff)
}

func LogError(stuff ...interface{}) {
	_log(os.Stderr, stuff)
}

func LogFatal(exitcode int, stuff ...interface{}) {
	_log(os.Stderr, stuff)
	os.Exit(exitcode)
}

func _log(w io.Writer, stuff ...interface{}) {
	s := make([]string, 0, len(stuff))
	for _, k := range stuff {
		s = append(s, fmt.Sprintf("%v", k))
	}
	ss := strings.Join(s, " ")
	fmt.Fprintln(w, ss)
}
