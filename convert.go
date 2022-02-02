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

	blocks := make(map[string][]*model.Block) // maps board ids to blocks
	exclude := make(map[string]struct{})      // block ids to exclude

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

				// filter out blocks for excluded boards
				if _, ok := exclude[block.RootID]; ok {
					exclude[block.ID] = struct{}{}
					continue
				}

				// if template mode then turn all boards into board templates
				if opts.TemplateMode && block.Type == "board" {
					block.Fields["isTemplate"] = true
					block.Fields["templateVer"] = ArchiveDestinationVersion
				}

				block.ModifiedBy = block.CreatedBy
				block.UpdateAt = utils.GetMillis()

				var boardID string
				if block.Type == "board" {
					boardID = block.ID
				} else {
					boardID = block.RootID
				}
				if blocks[boardID] == nil {
					blocks[boardID] = make([]*model.Block, 10)
				}
				blocks[boardID] = append(blocks[boardID], &block)
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

	removeOrphans(blocks)

	if opts.ShowImageInfo {
		showImageInfo(blocks)
		return nil
	}

	return CreateArchive(blocks, opts)
}

func showImageInfo(blocks map[string][]*model.Block) {
	for _, boardBlocks := range blocks {
		for _, block := range boardBlocks {
			if block == nil || block.Type != "image" {
				continue
			}

			imgFilename, _ := extractImageFilename(block)
			board, card := getBoardAndCard(block, boardBlocks)
			if board == nil || card == nil || !isInContentOrder(card, block) {
				LogDebug("orphan image: ", imgFilename)
				continue
			}
			fmt.Printf("boardId: %s, boardTitle: `%s`, cardID: %s, cardTitle: `%s`, filename: %s\n",
				board.ID, board.Title, card.ID, card.Title, imgFilename)
		}
	}
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
