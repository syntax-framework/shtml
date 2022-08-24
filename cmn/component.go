package cmn

// ComponentParamType representa os tipos de dados
type ComponentParamType uint

const (
  ParamTypeUnknown ComponentParamType = iota
  ParamTypeBool
  ParamTypeString
  ParamTypeArray
  ParamTypeFunc
  ParamTypeNumber // Go float64
  ParamTypeObject // Go Struct|Map, Javascript Object (JSON.parse() | JSON.stringify())
)

// ParamTypeNames os nomes válidos de tipos de parametros aceitos por componentes
var ParamTypeNames = map[string]ComponentParamType{
  "unknown":  ParamTypeUnknown,
  "bool":     ParamTypeBool,
  "string":   ParamTypeString,
  "array":    ParamTypeArray,
  "function": ParamTypeFunc,
  "number":   ParamTypeNumber,
  "object":   ParamTypeObject,
}

// ComponentParam representation of a parameter of a component
type ComponentParam struct {
  Name      string
  Type      ComponentParamType
  TypeName  string
  Required  bool
  IsClient  bool            // Indicates that it is a parameter for client side (Javascript)
  Reference *ComponentParam // When exposing the parameter to JS, it refers to a server parameter
}

type ComponentConfig struct {
  Params map[string]ComponentParam
}
