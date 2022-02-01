package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/mattermost/focalboard/server/model"
	"github.com/mattermost/focalboard/server/utils"
)

const (
	ArchiveSourceVersion = 1
)

type ConvertOptions struct {
	OutputFilename string
	DataDir        string
	TemplateMode   bool
	ShowImageInfo  bool
}

func Convert(archiveJSONL []byte, opts ConvertOptions) error {
	lineReader := bufio.NewReader(bytes.NewReader(archiveJSONL))

	// read the version info
	line, errRead := readLine(lineReader)
	if errRead != nil {
		return fmt.Errorf("error reading archive header: %w", errRead)
	}
	if err := checkVersion(line); err != nil {
		return err
	}

	blocks := make(map[string][]model.Block) // maps board ids to blocks
	exclude := make(map[string]struct{})     // block ids to exclude

	// read all the blocks
	lineNum := 1
	for {
		line, errRead := readLine(lineReader)
		if len(line) != 0 {
			var archiveLine model.ArchiveLine
			err := json.Unmarshal(line, &archiveLine)
			if err != nil {
				return fmt.Errorf("error parsing archive line %d: %w", lineNum, err)
			}
			switch archiveLine.Type {
			case "block":
				var block model.Block
				if err2 := json.Unmarshal(archiveLine.Data, &block); err2 != nil {
					return fmt.Errorf("invalid block in archive line %d: %w", lineNum, err2)
				}

				// in template mode we just want the non-template blocks, and we will turn
				// them into template blocks in the new archive.
				if opts.TemplateMode && isTemplate(block) {
					exclude[block.ID] = struct{}{}
					continue
				}

				// filter out excluded blocks and their children
				if _, ok := exclude[block.ParentID]; ok {
					exclude[block.ID] = struct{}{}
					continue
				}

				// if template mode then turn all boards into board templates
				if opts.TemplateMode && block.Type == "board" {
					block.Fields["isTemplate"] = true
					block.Fields["templateVer"] = ArchiveDestinationVersion
				}

				if opts.ShowImageInfo {
					if block.Type == "image" {
						imgFilename, _ := extractImageFilename(block)
						fmt.Printf("boardId: %s, filename: %s\n", block.RootID, imgFilename)
					}
					continue
				}

				block.ModifiedBy = "archiveconv"
				block.UpdateAt = utils.GetMillis()

				var boardID string
				if block.Type == "board" {
					boardID = block.ID
				} else {
					boardID = block.RootID
				}
				if blocks[boardID] == nil {
					blocks[boardID] = make([]model.Block, 10)
				}
				blocks[boardID] = append(blocks[boardID], block)
			default:
				return NewErrUnsupportedArchiveLineType(lineNum, archiveLine.Type)
			}
		}

		if errRead != nil {
			if errors.Is(errRead, io.EOF) {
				break
			}
			return fmt.Errorf("error reading archive line %d: %w", lineNum, errRead)
		}
		lineNum++
	}

	return CreateArchive(blocks, opts)
}

func readLine(r *bufio.Reader) ([]byte, error) {
	line, err := r.ReadBytes('\n')
	line = bytes.TrimSpace(line)
	return line, err
}

func checkVersion(line []byte) error {
	var header model.ArchiveHeader
	if err := json.Unmarshal(line, &header); err != nil {
		return fmt.Errorf("cannot read archive version: %w", err)
	}
	if header.Version != ArchiveSourceVersion {
		return NewErrUnsupportedArchiveVersion(header.Version, ArchiveSourceVersion)
	}
	return nil
}
