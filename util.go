package main

import "github.com/mattermost/focalboard/server/model"

func getBoardAndCard(block *model.Block, blocks []*model.Block) (board *model.Block, card *model.Block) {
	if block.Type == "board" {
		return block, nil
	}

	if block.Type == "card" {
		return getBlockByID(block.RootID, blocks), block
	}

	board = getBlockByID(block.RootID, blocks)
	card = getCard(block, blocks)
	return board, card
}

func getCard(block *model.Block, blocks []*model.Block) *model.Block {
	if block.Type == "board" {
		return nil
	}

	if block.Type == "card" {
		return block
	}

	b := block
	for {
		parent := getBlockByID(b.ParentID, blocks)
		if parent == nil {
			return nil
		}
		if parent.Type == "card" {
			return parent
		}
		b = parent
	}
}

func getBlockByID(blockID string, blocks []*model.Block) *model.Block {
	for _, b := range blocks {
		if b != nil && b.ID == blockID {
			return b
		}
	}
	return nil
}

func isInContentOrder(card *model.Block, block *model.Block) bool {
	if card == nil || block == nil {
		return false
	}

	co, ok := card.Fields["contentOrder"]
	if !ok {
		return false
	}

	contentOrder, ok := co.([]interface{})
	if !ok {
		LogFatal(-1, "invalid contentOrder for card id: ", card.ID)
	}

	for _, s := range contentOrder {
		id, ok := s.(string)
		if !ok {
			LogFatal(-1, "invalid contentOrder string for card id: ", card.ID)
		}

		if id == block.ID {
			return true
		}
	}
	return false
}

func removeOrphans(blocks map[string][]*model.Block) {
	for _, boardBlocks := range blocks {
		for i, block := range boardBlocks {
			if block == nil || block.Type == "board" || block.Type == "card" || block.Type == "view" {
				continue
			}

			card := getCard(block, boardBlocks)
			if card != nil && !isInContentOrder(card, block) {
				boardBlocks[i] = nil
			}
		}
	}
}
