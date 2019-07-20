package markdown

import (
	"fmt"
	"io"
	"os"

	"github.com/russross/blackfriday"
)

func Render(source string, lineWidth int, leftPad int) []byte {
	renderer := newRenderer(lineWidth, leftPad)

	// astRenderer, err := newAstRenderer()
	// if err != nil {
	// 	panic(err)
	// }
	// blackfriday.Run([]byte(source), blackfriday.WithRenderer(astRenderer))

	return blackfriday.Run([]byte(source), blackfriday.WithRenderer(renderer))
}

var _ blackfriday.Renderer = &astRenderer{}

type astRenderer struct {
	f *os.File
}

func newAstRenderer() (*astRenderer, error) {
	f, err := os.OpenFile("/tmp/ast.puml", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, err
	}

	return &astRenderer{f: f}, nil
}

func (a *astRenderer) RenderNode(w io.Writer, node *blackfriday.Node, entering bool) blackfriday.WalkStatus {
	if entering {
		for child := node.FirstChild; child != nil; child = child.Next {
			_, _ = fmt.Fprintf(a.f, "%s --> %s\n", node.Type.String(), child.Type.String())
		}
	}

	return blackfriday.GoToNext
}

func (a *astRenderer) RenderHeader(w io.Writer, ast *blackfriday.Node) {
	// _, _ = fmt.Fprintln(a.f, "@startuml")
	_, _ = fmt.Fprintln(a.f, "(*) --> Document")
}

func (a *astRenderer) RenderFooter(w io.Writer, ast *blackfriday.Node) {
	// _, _ = fmt.Fprintln(a.f, "@enduml")
	_ = a.f.Close()
}
