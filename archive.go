package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/mattermost/focalboard/server/model"
)

const (
	ArchiveDestinationVersion = 2
)

var (
	newline = []byte{'\n'}
)

func CreateArchive(blocks map[string][]*model.Block, opts ConvertOptions) (errResult error) {
	file, err := os.Create(opts.OutputFilename)
	if err != nil {
		return fmt.Errorf("cannot create archive %s: %w", opts.OutputFilename, err)
	}

	// wrap the file in a zip writer.
	zw := zip.NewWriter(file)
	defer func() {
		errResult = zw.Close()
	}()

	if err := writeArchiveVersion(zw); err != nil {
		return err
	}

	for boardID, boardBlocks := range blocks {
		if err := writeBoard(zw, boardID, boardBlocks, opts); err != nil {
			return err
		}
	}
	return nil
}

// writeArchiveVersion writes a version file to the zip.
func writeArchiveVersion(zw *zip.Writer) error {
	archiveHeader := model.ArchiveHeader{
		Version: ArchiveDestinationVersion,
		Date:    model.GetMillis(),
	}
	b, _ := json.Marshal(&archiveHeader)

	w, err := zw.Create("version.json")
	if err != nil {
		return fmt.Errorf("cannot write archive header: %w", err)
	}

	if _, err := w.Write(b); err != nil {
		return fmt.Errorf("cannot write archive header: %w", err)
	}
	return nil
}

func writeBoard(zw *zip.Writer, boardID string, boardBlocks []*model.Block, opts ConvertOptions) error {
	// create a directory per board
	w, err := zw.Create(boardID + "/board.jsonl")
	if err != nil {
		return err
	}

	// first pass to write the board.jsonl
	for _, block := range boardBlocks {
		if block == nil {
			continue
		}
		if err := writeArchiveBlockLine(w, block); err != nil {
			return fmt.Errorf("error writing to archive: %w", err)
		}
	}

	// second pass to write out image files
	for _, block := range boardBlocks {
		if block == nil || block.Type != "image" {
			continue
		}

		filename, err := extractImageFilename(block)
		if err != nil {
			return err
		}

		if err := writeArchiveFile(zw, filename, boardID, opts); err != nil {
			return fmt.Errorf("error writing to archive: %w", err)
		}
	}
	return nil
}

// writeArchiveBlockLine writes a single block to the archive.
func writeArchiveBlockLine(w io.Writer, block *model.Block) error {
	b, err := json.Marshal(&block)
	if err != nil {
		return err
	}
	line := model.ArchiveLine{
		Type: "block",
		Data: b,
	}

	b, err = json.Marshal(&line)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	if err != nil {
		return err
	}

	// jsonl files need a newline
	_, err = w.Write(newline)
	return err
}

func extractImageFilename(imageBlock *model.Block) (string, error) {
	f, ok := imageBlock.Fields["fileId"]
	if !ok {
		return "", ErrInvalidImageBlock
	}

	filename, ok := f.(string)
	if !ok {
		return "", ErrInvalidImageBlock
	}
	return filename, nil
}

// writeArchiveFile writes a single file to the archive.
func writeArchiveFile(zw *zip.Writer, filename string, boardID string, opts ConvertOptions) error {
	filespec := filepath.Join(opts.DataDir, boardID, filename)
	src, err := os.Open(filespec)
	if err != nil {
		LogError("cannot read image:", filespec, ":", err)
		return nil
	}
	defer src.Close()

	dest, err := zw.Create(boardID + "/" + filename)
	if err != nil {
		return err
	}

	_, err = io.Copy(dest, src)
	return err
}

func isTemplate(block model.Block) bool {
	if b, ok := block.Fields["isTemplate"]; ok {
		if val, ok := b.(bool); ok && val {
			return true
		}
	}
	return false
}
