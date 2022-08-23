package jsc

import (
	"github.com/syntax-framework/shtml/cmn"
	"github.com/syntax-framework/shtml/sht"
)

// Javascript Representa um recurso necessário por um componente
type Javascript struct {
	Code   string
	Params []cmn.ComponentParam
}

// HtmlEventsPush list of events that are enabled by default to push to server
var HtmlEventsPush = sht.CreateBoolMap([]string{
	// Form Event BindedAttributes
	"onblur", "onchange", "oncontextmenu", "onfocus", "oninput", "oninvalid", "onreset", "onsearch", "onselect", "onsubmit",
	// Mouse Event BindedAttributes
	"onclick", "ondblclick", "onmousedown", "onmousemove", "onmouseout", "onmouseover", "onmouseup", "onwheel",
})

// ClientComponentFields The methods and attributes that are part of the structure of a JS component
var ClientComponentFields = []string{
	// used now
	"api",          // Object - Allows the component to expose an API for external consumption. see ref
	"onMount",      // () => void - A method that runs after initial render and elements have been mounted
	"beforeUpdate", // () => void -
	"afterUpdate",  // () => void -
	"onCleanup",    // () => void - A cleanup method that executes on disposal or recalculation of the current reactive scope.
	"onDestroy",    // () => void - After unmount
	"onConnect",    // () => void - Invoked when the component has connected/reconnected to the server
	"onDisconnect", // () => void - Executed when the component is disconnected from the server
	"onError",      // (err: any) => void - Error handler method that executes when child scope errors
	// for future use
}

var ClientComponentFieldsMap = sht.CreateBoolMap(ClientComponentFields)

// ClientInvalidParamsAndRefs reserved variable names, cannot be used in parameters or references.
// The prefix "_$" is also not allowed.
var ClientInvalidParamsAndRefs = sht.CreateBoolMap(append([]string{
	"STX", "$", "push", "watch",
}, ClientComponentFields...))
