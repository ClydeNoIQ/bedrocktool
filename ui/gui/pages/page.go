package pages

import (
	"gioui.org/layout"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"github.com/bedrock-tool/bedrocktool/ui/messages"
)

type HandlerFunc = func(data interface{}) messages.Message

type Page interface {
	ID() string
	Actions() []component.AppBarAction
	Overflow() []component.OverflowAction
	Layout(gtx layout.Context, th *material.Theme) layout.Dimensions
	NavItem() component.NavItem
	messages.Handler
}

var Pages = map[string]func(*Router) Page{}

func Register(name string, fun func(*Router) Page) {
	Pages[name] = fun
}
