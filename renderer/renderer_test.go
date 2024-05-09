package renderer_test

import (
	"bytes"
	"testing"

	"github.com/go-test/deep"
	"github.com/slack-go/slack"
	"github.com/yuin/goldmark"

	"github.com/vslinko/mdblockkit/renderer"
)

func TestBoldInsideItalic(t *testing.T) {
	var blocks slack.Blocks

	markdown := goldmark.New(
		goldmark.WithRenderer(
			renderer.CreateMyRenderer(&blocks),
		),
	)

	var buf bytes.Buffer
	err := markdown.Convert([]byte("some _italic and **bold** text_ test"), &buf)
	if err != nil {
		t.Fatal(err)
	}

	if diff := deep.Equal(blocks, slack.Blocks{
		BlockSet: []slack.Block{
			&slack.RichTextBlock{
				Type: "rich_text",
				Elements: []slack.RichTextElement{
					&slack.RichTextSection{
						Type: "rich_text_section",
						Elements: []slack.RichTextSectionElement{
							&slack.RichTextSectionTextElement{
								Type: "text",
								Text: "some ",
							},
							&slack.RichTextSectionTextElement{
								Type: "text",
								Text: "italic and ",
								Style: &slack.RichTextSectionTextStyle{
									Italic: true,
								},
							},
							&slack.RichTextSectionTextElement{
								Type: "text",
								Text: "bold",
								Style: &slack.RichTextSectionTextStyle{
									Italic: true,
									Bold:   true,
								},
							},
							&slack.RichTextSectionTextElement{
								Type: "text",
								Text: " text",
								Style: &slack.RichTextSectionTextStyle{
									Italic: true,
								},
							},
							&slack.RichTextSectionTextElement{
								Type: "text",
								Text: " test",
							},
							&slack.RichTextSectionTextElement{
								Type: "text",
								Text: "\n\n",
							},
						},
					},
				},
			},
		},
	}); diff != nil {
		t.Error(diff)
	}
}
