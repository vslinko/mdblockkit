package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/slack-go/slack"
	"github.com/yuin/goldmark"

	"github.com/vslinko/mdblockkit/renderer"
)

type Result struct {
	Blocks slack.Blocks `json:"blocks"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: mdblockkit MD_FILE")
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	result := Result{}

	markdown := goldmark.New(
		goldmark.WithRenderer(
			renderer.CreateMyRenderer(&result.Blocks),
		),
	)

	var buf bytes.Buffer
	err = markdown.Convert(data, &buf)
	if err != nil {
		panic(err)
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		panic(err)
	}

	os.Stdout.Write(jsonData)
	os.Stdout.Write([]byte("\n"))
}
