package renderer

import (
	"errors"
	"io"

	"github.com/slack-go/slack"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
)

type MyRenderer struct {
	blocks          *slack.Blocks
	insideHeader    bool
	blockquoteLevel int
	listLevel       int
}

func CreateMyRenderer(blocks *slack.Blocks) *MyRenderer {
	return &MyRenderer{
		blocks:          blocks,
		insideHeader:    false,
		blockquoteLevel: 0,
		listLevel:       0,
	}
}

func (r MyRenderer) Render(w io.Writer, source []byte, node ast.Node) error {
	blocks, err := r.RenderDocument(w, source, node)
	if err != nil {
		return err
	}

	r.blocks.BlockSet = append(r.blocks.BlockSet, blocks...)

	return nil
}

func (r MyRenderer) RenderDocument(w io.Writer, source []byte, node ast.Node) ([]slack.Block, error) {
	if err := r.AssertKind(node, ast.KindDocument); err != nil {
		return nil, err
	}

	blocks, err := r.RenderBlocks(w, source, node)
	if err != nil {
		return nil, err
	}

	return blocks, nil
}

func (r MyRenderer) RenderBlocks(w io.Writer, source []byte, node ast.Node) ([]slack.Block, error) {
	blocks := []slack.Block{}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		childBlocks, err := r.RenderBlock(w, source, child)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, childBlocks...)
	}

	return blocks, nil
}

func (r MyRenderer) RenderBlock(w io.Writer, source []byte, node ast.Node) ([]slack.Block, error) {
	switch node.Kind() {
	case ast.KindHeading:
		return r.RenderHeading(w, source, node)
	case ast.KindParagraph:
		return r.RenderParagraph(w, source, node)
	case ast.KindTextBlock:
		return r.RenderTextBlock(w, source, node)
	case ast.KindBlockquote:
		return r.RenderBlockquote(w, source, node)
	case ast.KindList:
		return r.RenderList(w, source, node)
	default:
		return nil, errors.New("unsupported node type: " + node.Kind().String())
	}
}

func (r MyRenderer) RenderList(w io.Writer, source []byte, node ast.Node) ([]slack.Block, error) {
	if err := r.AssertKind(node, ast.KindList); err != nil {
		return nil, err
	}

	r.listLevel++

	listNode := node.(*ast.List)

	style := slack.RTEListBullet
	if listNode.IsOrdered() {
		style = slack.RTEListOrdered
	}

	elements := []slack.RichTextElement{}
	subElements := []slack.RichTextElement{}

	for child := listNode.FirstChild(); child != nil; child = child.NextSibling() {
		block, err := r.RenderBlocks(w, source, child)
		if err != nil {
			return nil, err
		}
		for _, b := range block {
			rtb := b.(*slack.RichTextBlock)
			for _, element := range rtb.Elements {
				switch element.RichTextElementType() {
				case slack.RTESection:
					section := element.(*slack.RichTextSection)
					subElements = append(subElements, section)
				case slack.RTEList:
					elements = append(elements, slack.NewRichTextList(
						style, r.listLevel-1, subElements...,
					))
					elements = append(elements, element.(slack.RichTextList))
					subElements = []slack.RichTextElement{}
				default:
					return nil, errors.New("unsupported element type: " + string(element.RichTextElementType()))
				}
			}
		}
	}

	if len(subElements) > 0 {
		elements = append(elements, *slack.NewRichTextList(
			style, r.listLevel-1, subElements...,
		))
	}

	r.listLevel--

	return []slack.Block{
		slack.NewRichTextBlock("", elements...),
	}, nil
}

func (r MyRenderer) RenderBlockquote(w io.Writer, source []byte, node ast.Node) ([]slack.Block, error) {
	if err := r.AssertKind(node, ast.KindBlockquote); err != nil {
		return nil, err
	}

	if r.blockquoteLevel > 0 {
		return nil, errors.New("nested blockquotes are not supported")
	}

	r.blockquoteLevel++
	blocks, err := r.RenderBlocks(w, source, node)
	if err != nil {
		return nil, err
	}
	r.blockquoteLevel--

	elements := []slack.RichTextSectionElement{}
	for _, block := range blocks {
		rtb := block.(*slack.RichTextBlock)
		for _, element := range rtb.Elements {
			switch element.RichTextElementType() {
			case slack.RTESection:
				section := element.(*slack.RichTextSection)
				elements = append(elements, section.Elements...)
			default:
				return nil, errors.New("unsupported element type: " + string(element.RichTextElementType()))
			}
		}
	}

	section := &slack.RichTextQuote{
		Type:     slack.RTEQuote,
		Elements: elements,
	}

	richTextBlock := slack.NewRichTextBlock("", section)

	return []slack.Block{
		richTextBlock,
	}, nil
}

func (r MyRenderer) RenderTextBlock(w io.Writer, source []byte, node ast.Node) ([]slack.Block, error) {
	if err := r.AssertKind(node, ast.KindTextBlock); err != nil {
		return nil, err
	}

	elements, err := r.RenderRichTextSectionElements(w, source, node)
	if err != nil {
		return nil, err
	}

	elements = append(elements, slack.NewRichTextSectionTextElement("\n\n", nil))

	richTextBlock := slack.NewRichTextBlock("", slack.NewRichTextSection(elements...))

	return []slack.Block{
		richTextBlock,
	}, nil
}

func (r MyRenderer) RenderParagraph(w io.Writer, source []byte, node ast.Node) ([]slack.Block, error) {
	if err := r.AssertKind(node, ast.KindParagraph); err != nil {
		return nil, err
	}

	elements, err := r.RenderRichTextSectionElements(w, source, node)
	if err != nil {
		return nil, err
	}

	elements = append(elements, slack.NewRichTextSectionTextElement("\n\n", nil))

	richTextBlock := slack.NewRichTextBlock("", slack.NewRichTextSection(elements...))

	return []slack.Block{
		richTextBlock,
	}, nil
}

func (r MyRenderer) RenderRichTextSectionElements(w io.Writer, source []byte, node ast.Node) ([]slack.RichTextSectionElement, error) {
	elements := []slack.RichTextSectionElement{}

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		childElements, err := r.RenderRichTextSectionElement(w, source, child)
		if err != nil {
			return nil, err
		}
		elements = append(elements, childElements...)
	}

	return elements, nil
}

func (r MyRenderer) RenderRichTextSectionElement(w io.Writer, source []byte, node ast.Node) ([]slack.RichTextSectionElement, error) {
	switch node.Kind() {
	case ast.KindText:
		return r.RenderText(w, source, node)
	case ast.KindEmphasis:
		return r.RenderEmphasis(w, source, node)
	case ast.KindLink:
		return r.RenderLink(w, source, node)
	default:
		return nil, errors.New("unsupported node type: " + node.Kind().String())
	}
}

func (r MyRenderer) RenderEmphasis(w io.Writer, source []byte, node ast.Node) ([]slack.RichTextSectionElement, error) {
	if err := r.AssertKind(node, ast.KindEmphasis); err != nil {
		return nil, err
	}

	emphasisNode := node.(*ast.Emphasis)
	style := slack.RichTextSectionTextStyle{}

	if r.insideHeader || emphasisNode.Level == 2 {
		style.Bold = true
	}
	if emphasisNode.Level == 1 {
		style.Italic = true
	}

	text := string(node.Text(source))

	return []slack.RichTextSectionElement{
		slack.NewRichTextSectionTextElement(text, &style),
	}, nil
}

func (r MyRenderer) RenderLink(w io.Writer, source []byte, node ast.Node) ([]slack.RichTextSectionElement, error) {
	if err := r.AssertKind(node, ast.KindLink); err != nil {
		return nil, err
	}

	linkNode := node.(*ast.Link)

	var style *slack.RichTextSectionTextStyle
	if r.insideHeader {
		style = &slack.RichTextSectionTextStyle{
			Bold: true,
		}
	}

	text := string(node.Text(source))

	return []slack.RichTextSectionElement{
		slack.NewRichTextSectionLinkElement(string(linkNode.Destination), text, style),
	}, nil
}

func (r MyRenderer) RenderText(w io.Writer, source []byte, node ast.Node) ([]slack.RichTextSectionElement, error) {
	if err := r.AssertKind(node, ast.KindText); err != nil {
		return nil, err
	}

	textNode := node.(*ast.Text)

	var style *slack.RichTextSectionTextStyle
	if r.insideHeader {
		style = &slack.RichTextSectionTextStyle{
			Bold: true,
		}
	}

	text := string(node.Text(source))

	if textNode.SoftLineBreak() || textNode.HardLineBreak() {
		text += "\n"
	}

	return []slack.RichTextSectionElement{
		slack.NewRichTextSectionTextElement(text, style),
	}, nil
}

func (r MyRenderer) RenderHeading(w io.Writer, source []byte, node ast.Node) ([]slack.Block, error) {
	if err := r.AssertKind(node, ast.KindHeading); err != nil {
		return nil, err
	}

	prevInsideHeader := r.insideHeader
	r.insideHeader = true
	elements, err := r.RenderRichTextSectionElements(w, source, node)
	if err != nil {
		return nil, err
	}
	r.insideHeader = prevInsideHeader

	elements = append(elements, slack.NewRichTextSectionTextElement("\n\n", nil))

	richTextBlock := slack.NewRichTextBlock("", slack.NewRichTextSection(elements...))

	return []slack.Block{
		richTextBlock,
	}, nil
}

func (r MyRenderer) AssertKind(node ast.Node, kind ast.NodeKind) error {
	if node.Kind() != kind {
		return errors.New("ast is not a " + kind.String())
	}

	return nil
}

func (r MyRenderer) AddOptions(opts ...renderer.Option) {
}
