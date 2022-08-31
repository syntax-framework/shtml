package jsc

import (
	"github.com/syntax-framework/shtml/cmn"
	"github.com/syntax-framework/shtml/sht"
)

// Javascript Represents a resource needed by a component
type Javascript struct {
	Content         string
	ComponentParams []cmn.ComponentParam
}

// HtmlEventsPush list of events that are enabled by default to push to server
var HtmlEventsPush = sht.CreateBoolMap([]string{
	// Form Event AttributeNames
	"onblur", "onchange", "oncontextmenu", "onfocus", "oninput", "oninvalid", "onreset", "onsearch", "onselect", "onsubmit",
	// Mouse Event AttributeNames
	"onclick", "ondblclick", "onmousedown", "onmousemove", "onmouseout", "onmouseover", "onmouseup", "onwheel",
})

// ClientLifeCycleMap The methods and attributes that are part of the structure of a JS component
var ClientLifeCycleMap = map[string]string{
	"OnMount":      "a", // `() => void` A method that runs after initial render and elements have been mounted
	"BeforeUpdate": "b", // `() => void`
	"AfterUpdate":  "c", // `() => void`
	"BeforeRender": "d", // `() => void`
	"AfterRender":  "e", // `() => void`
	"OnDestroy":    "f", // `() => void` Before unmount
	"OnConnect":    "g", // `() => void` Invoked when the component has connected/reconnected to the server
	"OnDisconnect": "h", // `() => void` Executed when the component is disconnected from the server
	"OnEvent":      "i", // `(event) => bool`
	"OnError":      "j", // `(trace: string, err: any) => void` Error handler method that executes when child scope errors
}

// ClientInvalidParamsAndRefs reserved variable names, cannot be used in parameters or references.
// The prefix "_$" is also not allowed.
var ClientInvalidParamsAndRefs = sht.CreateBoolMap([]string{
	"STX", "$", "push", "watch", "tick",
	// from ClientLifeCycleMap
	"OnMount", "BeforeUpdate", "AfterUpdate", "BeforeRender", "AfterRender", "OnDestroy", "OnConnect", "OnDisconnect",
	"OnError", "OnEvent",
})
