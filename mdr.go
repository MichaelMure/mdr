package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/MichaelMure/go-term-markdown"
	"github.com/MichaelMure/gocui"
	"github.com/pkg/errors"
)

const padding = 4

func main() {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		exitError(errors.Wrap(err, "error starting the interactive UI"))
	}
	defer g.Close()

	ui, err := newUi(g)
	if err != nil {
		exitError(err)
	}

	err = ui.loadFile("example.md")
	if err != nil {
		exitError(errors.Wrap(err, "error reading file"))
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		exitError(err)
	}
}

func exitError(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

const renderView = "render"

type ui struct {
	raw string
	// current width of the view
	width   int
	XOffset int
	YOffset int

	// number of lines in the rendered markdown
	lines int
}

func newUi(g *gocui.Gui) (*ui, error) {
	result := &ui{
		width: -1,
	}

	g.SetManagerFunc(result.layout)

	// Quit
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, result.quit); err != nil {
		return nil, err
	}
	if err := g.SetKeybinding("", 'q', gocui.ModNone, result.quit); err != nil {
		return nil, err
	}

	// Up
	if err := g.SetKeybinding("", 'k', gocui.ModNone, result.up); err != nil {
		return nil, err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, result.up); err != nil {
		return nil, err
	}
	// Down
	if err := g.SetKeybinding("", 'j', gocui.ModNone, result.down); err != nil {
		return nil, err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, result.down); err != nil {
		return nil, err
	}

	// PageUp
	if err := g.SetKeybinding("", gocui.KeyPgup, gocui.ModNone, result.pageUp); err != nil {
		return nil, err
	}
	// PageDown
	if err := g.SetKeybinding("", gocui.KeyPgdn, gocui.ModNone, result.pageDown); err != nil {
		return nil, err
	}

	// // Left
	// if err := g.SetKeybinding("", 'h', gocui.ModNone, result.left); err != nil {
	// 	return nil, err
	// }
	// if err := g.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, result.left); err != nil {
	// 	return nil, err
	// }
	// // Right
	// if err := g.SetKeybinding("", 'l', gocui.ModNone, result.right); err != nil {
	// 	return nil, err
	// }
	// if err := g.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone, result.right); err != nil {
	// 	return nil, err
	// }

	return result, nil
}

func (ui *ui) loadFile(path string) error {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	ui.raw = string(raw)
	ui.width = -1
	return nil
}

func (ui *ui) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	v, err := g.SetView(renderView, ui.XOffset, -ui.YOffset, maxX, maxY)
	if err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Frame = false
		v.Wrap = false
	}

	if len(ui.raw) > 0 && ui.width != maxX {
		ui.width = maxX
		v.Clear()
		_, _ = v.Write(ui.render(g))
	}

	return nil
}

func (ui *ui) render(g *gocui.Gui) []byte {
	maxX, _ := g.Size()
	rendered := markdown.Render(ui.raw, maxX-1-padding, padding)
	ui.lines = 0
	for _, b := range rendered {
		if b == '\n' {
			ui.lines++
		}
	}
	return rendered
}

func (ui *ui) quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func (ui *ui) up(g *gocui.Gui, v *gocui.View) error {
	ui.YOffset -= 1
	ui.YOffset = max(ui.YOffset, 0)
	return nil
}

func (ui *ui) down(g *gocui.Gui, v *gocui.View) error {
	_, maxY := g.Size()
	ui.YOffset += 1
	ui.YOffset = min(ui.YOffset, ui.lines-maxY+1)
	ui.YOffset = max(ui.YOffset, 0)
	return nil
}

func (ui *ui) pageUp(g *gocui.Gui, v *gocui.View) error {
	_, maxY := g.Size()
	ui.YOffset -= maxY / 2
	ui.YOffset = max(ui.YOffset, 0)
	return nil
}

func (ui *ui) pageDown(g *gocui.Gui, v *gocui.View) error {
	_, maxY := g.Size()
	ui.YOffset += maxY / 2
	ui.YOffset = min(ui.YOffset, ui.lines-maxY+1)
	ui.YOffset = max(ui.YOffset, 0)
	return nil
}

// func (ui *ui) left(g *gocui.Gui, v *gocui.View) error {
// 	ui.XOffset += 1
// 	return nil
// }
//
// func (ui *ui) right(g *gocui.Gui, v *gocui.View) error {
// 	ui.XOffset -= 1
// 	return nil
// }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
