package ast

import (
    "bytes"
    "regexp"
    "strings"

    "github.com/yuin/goldmark"
    gmast "github.com/yuin/goldmark/ast"
    "github.com/yuin/goldmark/extension"
    "github.com/yuin/goldmark/text"
)

type SpecChunk struct {
    Heading string
    Domain  string
    Content string
}

func ClassifyHeading(text string) string {
    text = strings.ToLower(text)
    dbRegex := regexp.MustCompile(`(database|models|schema|entities|table)`)
    apiRegex := regexp.MustCompile(`(api|routes|endpoints|controllers|views)`)
    if dbRegex.MatchString(text) {
        return "data_schema"
    }
    if apiRegex.MatchString(text) {
        return "api_contract"
    }
    return "system_prose"
}

func ParseSpec(content []byte) ([]SpecChunk, error) {
    md := goldmark.New(
        goldmark.WithExtensions(extension.Table),
    )
    reader := text.NewReader(content)
    doc := md.Parser().Parse(reader)

    var chunks []SpecChunk
    var currentChunk *SpecChunk

    err := gmast.Walk(doc, func(n gmast.Node, entering bool) (gmast.WalkStatus, error) {
        if !entering {
            return gmast.WalkContinue, nil
        }
        if n.Kind() == gmast.KindHeading {
            headingNode := n.(*gmast.Heading)
            var headingText bytes.Buffer
            for child := headingNode.FirstChild(); child != nil; child = child.NextSibling() {
                if child.Kind() == gmast.KindText {
                    headingText.Write(child.(*gmast.Text).Segment.Value(content))
                }
            }
            hStr := strings.TrimSpace(headingText.String())
            domain := ClassifyHeading(hStr)
            if currentChunk != nil {
                chunks = append(chunks, *currentChunk)
            }
            currentChunk = &SpecChunk{
                Heading: hStr,
                Domain:  domain,
                Content: "",
            }
        } else if currentChunk != nil && n.Type() == gmast.TypeBlock {
            lines := n.Lines()
            for i := 0; i < lines.Len(); i++ {
                line := lines.At(i)
                currentChunk.Content += string(line.Value(content)) + "\n"
            }
        }
        return gmast.WalkContinue, nil
    })

    if err != nil {
        return nil, err
    }
    if currentChunk != nil {
        chunks = append(chunks, *currentChunk)
    }
    return chunks, nil
}
