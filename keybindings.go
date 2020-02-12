package main

import "github.com/awesome-gocui/gocui"

type keybinding struct {
	viewName string
	key      interface{}
	mod      gocui.Modifier
	handler  func(*gocui.Gui, *gocui.View) error
}

func (k *keybinding) Register(g *gocui.Gui) error {
	return g.SetKeybinding(k.viewName, k.key, k.mod, k.handler)
}
