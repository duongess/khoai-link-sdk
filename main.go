package main

import (
	"context"
	"log"

	"github.com/duongess/khoai-link-sdk/khoailinksdk"
	"github.com/duongess/khoai-link-sdk/types"
)

func main() {
	node := khoailinksdk.New("http://127.0.0.1:8001")

	// 1. Dang ky cac Task truoc
	node.RegisterTaskHandler("read_excel_row", func(ctx context.Context, rawInput types.TaskInput) (types.TaskOutput, error) {
		// Logic doc file Excel hoac DB
		return map[string]any{
			"lot_id": "LOT_999",
			"temp":   "85.5",
		}, nil
	})

	node.RegisterTaskHandler("check_temp_threshold", types.TaskHandler(func(ctx context.Context, rawInput types.TaskInput) (types.TaskOutput, error) {
		// Logic kiem tra nguong
		return map[string]any{
			"is_alert": "true",
		}, nil
	}))

	// 2. Start node
	if err := node.Start(); err != nil {
		log.Panicln("node error:", err)
	}
}
