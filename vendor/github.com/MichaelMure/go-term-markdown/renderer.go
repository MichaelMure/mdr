package markdown

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/MichaelMure/go-term-text"
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/fatih/color"
	"github.com/kyokomi/emoji"
	"github.com/russross/blackfriday"
)

/*

Here are the possible cases for the AST. You can render it using PlantUML.

@startuml

(*) --> Document
BlockQuote --> BlockQuote
BlockQuote --> CodeBlock
BlockQuote --> List
BlockQuote --> Paragraph
Del --> Emph
Del --> Strong
Del --> Text
Document --> BlockQuote
Document --> CodeBlock
Document --> Heading
Document --> HorizontalRule
Document --> HTMLBlock
Document --> List
Document --> Paragraph
Document --> Table
Emph --> Text
Heading --> Code
Heading --> Del
Heading --> Emph
Heading --> HTMLSpan
Heading --> Image
Heading --> Link
Heading --> Strong
Heading --> Text
Image --> Text
Item --> List
Item --> Paragraph
Link --> Image
Link --> Text
List --> Item
Paragraph --> Code
Paragraph --> Del
Paragraph --> Emph
Paragraph --> Hardbreak
Paragraph --> HTMLSpan
Paragraph --> Image
Paragraph --> Link
Paragraph --> Strong
Paragraph --> Text
Strong --> Emph
Strong --> Text
TableBody --> TableRow
TableCell --> Code
TableCell --> Del
TableCell --> Emph
TableCell --> HTMLSpan
TableCell --> Image
TableCell --> Link
TableCell --> Strong
TableCell --> Text
TableHead --> TableRow
TableRow --> TableCell
Table --> TableBody
Table --> TableHead

@enduml

*/

var _ blackfriday.Renderer = &renderer{}

type renderer struct {
	// maximum line width allowed
	lineWidth int
	// constant left padding to apply
	leftPad int

	// all the custom left paddings, without the fixed space from leftPad
	padAccumulator []string

	// one-shot indent for the first line of the inline content
	indent string

	// for Heading, Paragraph and TableCell, accumulate the content of
	// the child nodes (Link, Text, Image, formatting ...). The result
	// is then rendered appropriately when exiting the node.
	inlineAccumulator strings.Builder

	// record and render the heading numbering
	headingNumbering headingNumbering

	blockQuoteLevel int

	table *tableRenderer
}

func newRenderer(lineWidth int, leftPad int) *renderer {
	return &renderer{
		lineWidth:      lineWidth,
		leftPad:        leftPad,
		padAccumulator: make([]string, 0, 10),
	}
}

func (r *renderer) pad() string {
	return strings.Repeat(" ", r.leftPad) + strings.Join(r.padAccumulator, "")
}

func (r *renderer) addPad(pad string) {
	r.padAccumulator = append(r.padAccumulator, pad)
}

func (r *renderer) popPad() {
	r.padAccumulator = r.padAccumulator[:len(r.padAccumulator)-1]
}

func (r *renderer) RenderNode(w io.Writer, node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
	// TODO: remove
	// fmt.Println(node.Type, string(node.Literal), entering)

	switch node.Type {
	case blackfriday.Document:
		// Nothing to do

	case blackfriday.BlockQuote:
		// set and remove a colored bar on the left
		if entering {
			r.blockQuoteLevel++
			r.addPad(quoteShade(r.blockQuoteLevel)("┃ "))
		} else {
			r.blockQuoteLevel--
			r.popPad()
		}

	case blackfriday.List:
		if !entering && node.Next != nil {
			if node.Next.Type != blackfriday.List && node.Parent.Type != blackfriday.Item {
				_, _ = fmt.Fprintln(w)
			}
		}

	case blackfriday.Item:
		// write the prefix, add a padding if needed, and let Paragraph handle the rest
		if entering {
			switch {
			// numbered list
			case node.ListData.ListFlags&blackfriday.ListTypeOrdered != 0:
				itemNumber := 1
				for prev := node.Prev; prev != nil; prev = prev.Prev {
					itemNumber++
				}
				prefix := fmt.Sprintf("%d. ", itemNumber)
				r.indent = r.pad() + Green(prefix)
				r.addPad(strings.Repeat(" ", text.WordLen(prefix)))

			// header of a definition
			case node.ListData.ListFlags&blackfriday.ListTypeTerm != 0:
				r.inlineAccumulator.WriteString(greenOn)

			// content of a definition
			case node.ListData.ListFlags&blackfriday.ListTypeDefinition != 0:
				r.addPad("  ")

			// no flags means it's the normal bullet point list
			default:
				r.indent = r.pad() + Green("• ")
				r.addPad("  ")
			}
		} else {
			switch {
			// numbered list
			case node.ListData.ListFlags&blackfriday.ListTypeOrdered != 0:
				r.popPad()

			// header of a definition
			case node.ListData.ListFlags&blackfriday.ListTypeTerm != 0:
				r.inlineAccumulator.WriteString(colorOff)

			// content of a definition
			case node.ListData.ListFlags&blackfriday.ListTypeDefinition != 0:
				r.popPad()
				_, _ = fmt.Fprintln(w)

			// no flags means it's the normal bullet point list
			default:
				r.popPad()
			}
		}

	case blackfriday.Paragraph:
		// on exiting, collect and format the accumulated content
		if !entering {
			content := r.inlineAccumulator.String()
			r.inlineAccumulator.Reset()

			var out string
			if r.indent != "" {
				out, _ = text.WrapWithPadIndent(content, r.lineWidth, r.indent, r.pad())
				r.indent = ""
			} else {
				out, _ = text.WrapWithPad(content, r.lineWidth, r.pad())
			}
			_, _ = fmt.Fprint(w, out, "\n")

			// extra line break in some cases
			if node.Next != nil {
				switch node.Next.Type {
				case blackfriday.Paragraph, blackfriday.Heading, blackfriday.HorizontalRule,
					blackfriday.CodeBlock, blackfriday.HTMLBlock:
					_, _ = fmt.Fprintln(w)
				}

				if node.Next.Type == blackfriday.List && node.Parent.Type != blackfriday.Item {
					_, _ = fmt.Fprintln(w)
				}
			}
		}

	case blackfriday.Heading:
		if !entering {
			content := r.inlineAccumulator.String()
			r.inlineAccumulator.Reset()

			// render the full line with the headingNumbering
			r.headingNumbering.Observe(node.Level)
			rendered := fmt.Sprintf("%s%s %s", r.pad(), r.headingNumbering.Render(), content)

			// wrap if needed
			wrapped, _ := text.Wrap(rendered, r.lineWidth)
			colored := headingShade(node.Level)(wrapped)
			_, _ = fmt.Fprintln(w, colored)

			// render the underline, if any
			if node.Level == 1 {
				_, _ = fmt.Fprintf(w, "%s%s\n", r.pad(), strings.Repeat("─", r.lineWidth-r.leftPad))
			}

			_, _ = fmt.Fprintln(w)
		}

	case blackfriday.HorizontalRule:
		_, _ = fmt.Fprintf(w, "%s%s\n\n", r.pad(), strings.Repeat("─", r.lineWidth-r.leftPad))

	case blackfriday.Emph:
		if entering {
			r.inlineAccumulator.WriteString(italicOn)
		} else {
			r.inlineAccumulator.WriteString(italicOff)
		}

	case blackfriday.Strong:
		if entering {
			r.inlineAccumulator.WriteString(boldOn)
		} else {
			r.inlineAccumulator.WriteString(boldOff)
		}

	case blackfriday.Del:
		if entering {
			r.inlineAccumulator.WriteString(crossedOutOn)
		} else {
			r.inlineAccumulator.WriteString(crossedOutOff)
		}

	case blackfriday.Link:
		r.inlineAccumulator.WriteString("[")
		r.inlineAccumulator.WriteString(string(node.FirstChild.Literal))
		r.inlineAccumulator.WriteString("](")
		r.inlineAccumulator.WriteString(Blue(string(node.LinkData.Destination)))
		if len(node.LinkData.Title) > 0 {
			r.inlineAccumulator.WriteString(" ")
			r.inlineAccumulator.WriteString(string(node.LinkData.Title))
		}
		r.inlineAccumulator.WriteString(")")
		return blackfriday.SkipChildren

	case blackfriday.Image:

	case blackfriday.Text:
		content := string(node.Literal)
		if shouldCleanText(node) {
			content = removeLineBreak(content)
		}
		// emoji support !
		emojed := emoji.Sprint(content)
		r.inlineAccumulator.WriteString(emojed)

	case blackfriday.HTMLBlock:
		content := Red(string(node.Literal))
		out, _ := text.WrapWithPad(content, r.lineWidth, r.pad())
		_, _ = fmt.Fprint(w, out, "\n\n")

	case blackfriday.CodeBlock:
		r.renderCodeBlock(w, node)

	case blackfriday.Softbreak:
		// not actually implemented in blackfriday
		r.inlineAccumulator.WriteString("\n")

	case blackfriday.Hardbreak:
		r.inlineAccumulator.WriteString("\n")

	case blackfriday.Code:
		r.inlineAccumulator.WriteString(BlueBgItalic(string(node.Literal)))

	case blackfriday.HTMLSpan:
		r.inlineAccumulator.WriteString(Red(string(node.Literal)))

	case blackfriday.Table:
		if entering {
			r.table = newTableRenderer()
		} else {
			r.table.Render(w, r.leftPad, r.lineWidth)
			r.table = nil
		}

	case blackfriday.TableCell:
		if !entering {
			content := r.inlineAccumulator.String()
			r.inlineAccumulator.Reset()

			if node.TableCellData.IsHeader {
				r.table.AddHeaderCell(content, node.TableCellData.Align)
			} else {
				r.table.AddBodyCell(content)
			}
		}

	case blackfriday.TableHead:
		// nothing to do

	case blackfriday.TableBody:
		// nothing to do

	case blackfriday.TableRow:
		if entering && node.Parent.Type == blackfriday.TableBody {
			r.table.NextBodyRow()
		}

	default:
		panic("Unknown node type " + node.Type.String())
	}

	return blackfriday.GoToNext
}

func (*renderer) RenderHeader(w io.Writer, ast *blackfriday.Node) {}

func (*renderer) RenderFooter(w io.Writer, ast *blackfriday.Node) {}

func (r *renderer) renderCodeBlock(w io.Writer, node *blackfriday.Node) {
	code := string(node.Literal)
	var lexer chroma.Lexer
	// try to get the lexer from the language tag if any
	if len(node.CodeBlockData.Info) > 0 {
		lexer = lexers.Get(string(node.CodeBlockData.Info))
	}
	// fallback on detection
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	// all failed :-(
	if lexer == nil {
		lexer = lexers.Fallback
	}
	// simplify the lexer output
	lexer = chroma.Coalesce(lexer)

	var formatter chroma.Formatter
	if color.NoColor {
		formatter = formatters.Fallback
	} else {
		formatter = formatters.TTY8
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		// Something failed, falling back to no highlight render
		r.renderFormattedCodeBlock(w, code)
		return
	}

	buf := &bytes.Buffer{}

	err = formatter.Format(buf, styles.Pygments, iterator)
	if err != nil {
		// Something failed, falling back to no highlight render
		r.renderFormattedCodeBlock(w, code)
		return
	}

	r.renderFormattedCodeBlock(w, buf.String())
}

func (r *renderer) renderFormattedCodeBlock(w io.Writer, code string) {
	// remove the trailing line break
	code = strings.TrimRight(code, "\n")

	r.addPad(GreenBold("┃ "))
	output, _ := text.WrapWithPad(code, r.lineWidth, r.pad())
	r.popPad()

	_, _ = fmt.Fprint(w, output)

	_, _ = fmt.Fprintf(w, "\n\n")
}

func removeLineBreak(text string) string {
	lines := strings.Split(text, "\n")

	if len(lines) <= 1 {
		return text
	}

	for i, l := range lines {
		switch i {
		case 0:
			lines[i] = strings.TrimRightFunc(l, unicode.IsSpace)
		case len(lines) - 1:
			lines[i] = strings.TrimLeftFunc(l, unicode.IsSpace)
		default:
			lines[i] = strings.TrimFunc(l, unicode.IsSpace)
		}
	}
	return strings.Join(lines, " ")
}

func shouldCleanText(node *blackfriday.Node) bool {
	for node != nil {
		switch node.Type {
		case blackfriday.BlockQuote:
			return false

		case blackfriday.Heading, blackfriday.Image, blackfriday.Link,
			blackfriday.TableCell, blackfriday.Document, blackfriday.Item:
			return true
		}

		node = node.Parent
	}

	panic("bad markdown document or missing case")
}
