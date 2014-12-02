// Package dom is a front-end MVC framework for use with http://gopherjs.org/.
// It uses a concept of aspects, which modify apperance (properties and styles),
// behavior (event listeners) and content (child elements and text nodes) of
// DOM elements. An aspect can be applied and reverted. A tree of aspects
// represents a tree of DOM elements in a dynamic fashion. By using the aspects
// of the bind package, one may bind to the data model and modify the DOM
// elements automatically according to changes of the model.
package dom

import (
	"fmt"

	"github.com/gopherjs/gopherjs/js"
)

// Aspect is the basic building block of the dom package. A DOM element can have
// many aspects that modify apperance, behavior and content.
type Aspect interface {
	Apply(node js.Object, p, r float64)
	Revert()
}

type groupAspect []Aspect

// Group combines multiple aspects into one.
func Group(aspects ...Aspect) Aspect {
	if len(aspects) == 1 {
		return aspects[0]
	}
	return groupAspect(aspects)
}

func (g groupAspect) Apply(node js.Object, p, r float64) {
	r2 := r / float64(len(g))
	for i, a := range g {
		a.Apply(node, p+r2*float64(i), r2)
	}
}

func (g groupAspect) Revert() {
	for _, a := range g {
		a.Revert()
	}
}

type nodeAspect struct {
	node  js.Object
	child Aspect
}

// Element creates a new DOM element and adds it when applied. It removes the
// element when reverted. The elem package provides helper functions to be used
// instead of this function in most cases.
func Element(tagName string, aspects ...Aspect) Aspect {
	return &nodeAspect{
		node:  js.Global.Get("document").Call("createElement", tagName),
		child: Group(aspects...),
	}
}

func (e *nodeAspect) Apply(node js.Object, p, r float64) {
	if !e.node.Get("previousSibling").IsNull() && e.node.Get("previousSibling").Get("gopherjsDomPosition").Float() > p {
		e.node.Call("remove")
	}
	if e.node.Get("parentNode").IsNull() {
		e.node.Set("gopherjsDomPosition", p)
		c := node.Get("firstChild")
		for !c.IsNull() && c.Get("gopherjsDomPosition").Float() < p {
			c = c.Get("nextSibling")
		}
		node.Call("insertBefore", e.node, c)
	}
	if e.child != nil {
		e.child.Apply(e.node, 0, 1)
	}
}

func (e *nodeAspect) Revert() {
	e.node.Call("remove")
}

// Text creates a new DOM text node and adds it when applied. It removes the node when reverted.
func Text(content string) Aspect {
	return &nodeAspect{
		node: js.Global.Get("document").Call("createTextNode", content),
	}
}

type setPropertyAspect struct {
	name  string
	value string
}

// SetProperty sets a string property when applied. It does NOT reset the
// property when reverted. The prop package provides helper functions to be used
// instead of this function in most cases.
func SetProperty(name string, value string) Aspect {
	return &setPropertyAspect{name: name, value: value}
}

func (a *setPropertyAspect) Apply(node js.Object, p, r float64) {
	if node.Get(a.name).Str() != a.value {
		node.Set(a.name, a.value)
	}
}

func (a *setPropertyAspect) Revert() {
	// no reset
}

type togglePropertyAspect struct {
	name string
	node js.Object
}

// ToggleProperty sets a boolean property to true when applied. It sets it to
// false when reverted. The prop package provides helper functions to be used
// instead of this function in most cases.
func ToggleProperty(name string) Aspect {
	return &togglePropertyAspect{name: name}
}

func (a *togglePropertyAspect) Apply(node js.Object, p, r float64) {
	a.node = node
	node.Set(a.name, true)
}

func (a *togglePropertyAspect) Revert() {
	a.node.Set(a.name, false)
}

type styleAspect struct {
	name  string
	value string
	style js.Object
}

// Style sets a style when applied. It removes the style when reverted. The
// style package provides helper functions to be used instead of this function
// in most cases.
func Style(name string, value string) Aspect {
	return &styleAspect{
		name:  name,
		value: value,
	}
}

func (a *styleAspect) Apply(node js.Object, p, r float64) {
	a.style = node.Get("style")
	a.style.Call("setProperty", a.name, a.value, "important")
}

func (a *styleAspect) Revert() {
	a.style.Call("removeProperty", a.name)
}

// Listener is a callback for DOM events.
type Listener func(c *EventContext)

// EventContext provides details about event.
type EventContext struct {
	// JavaScript's "this" in the event callback.
	Node js.Object
	// The first argument given to the event callback (usually the event object).
	Event js.Object
}

type eventAspect struct {
	eventType      string
	listener       func(event js.Object)
	preventDefault bool
	node           js.Object
}

// Event adds the given event listener when applied. It removes the listener
// when reverted. The event package provides helper functions to be used instead
// of this function in most cases.
func Event(eventType string, listener Listener) Aspect {
	var a *eventAspect
	a = &eventAspect{
		eventType: eventType,
		listener: func(event js.Object) {
			if a.preventDefault {
				event.Call("preventDefault")
			}
			go listener(&EventContext{
				Node:  a.node,
				Event: event,
			})
		},
		preventDefault: false,
	}
	return a
}

// PreventDefault calls event.preventDefault() when handling the given event.
func PreventDefault(aspect Aspect) Aspect {
	aspect.(*eventAspect).preventDefault = true
	return aspect
}

func (a *eventAspect) Apply(node js.Object, p, r float64) {
	if a.node == nil {
		a.node = node
		a.node.Call("addEventListener", a.eventType, a.listener)
	}
}

func (a *eventAspect) Revert() {
	if a.node != nil {
		a.node.Call("removeEventListener", a.eventType, a.listener)
		a.node = nil
	}
}

type debugAspect struct {
	msg interface{}
}

// Debug prints to the console when applied or removed.
func Debug(msg interface{}) Aspect {
	return &debugAspect{msg: msg}
}

func (a *debugAspect) Apply(node js.Object, p, r float64) {
	println("Apply:", fmt.Sprint(a.msg), node)
}

func (a *debugAspect) Revert() {
	println("Revert:", fmt.Sprint(a.msg))
}

// AddToBody applies the given aspects to the page's body element.
func AddToBody(aspects ...Aspect) {
	Group(aspects...).Apply(js.Global.Get("document").Get("body"), 0, 1)
}