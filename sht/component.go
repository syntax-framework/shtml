package sht

// ParamType representa os tipos de dados
type ParamType uint

const (
  ParamTypeUnknown ParamType = iota
  ParamTypeBool
  ParamTypeInt
  ParamTypeUint
  ParamTypeFloat
  ParamTypeString
  ParamTypeFunc
  ParamTypeMap
  ParamTypeArray
  ParamTypeStruct
)

// ParamTypeNames os nomes válidos de tipos de parametros aceitos por componentes
var ParamTypeNames = map[string]ParamType{
  "bool":   ParamTypeBool,
  "int":    ParamTypeInt,
  "uint":   ParamTypeUint,
  "float":  ParamTypeFloat,
  "string": ParamTypeString,
  "func":   ParamTypeFunc,
  "map":    ParamTypeMap,
  "array":  ParamTypeArray,
  "struct": ParamTypeStruct,
}

// Live interface de inter
type Live struct {
}

// ComponentParam representation of a parameter of a component
type ComponentParam struct {
  Name      string
  Type      ParamType
  TypeName  string
  Required  bool
  IsJs      bool            // Indicates that it is a parameter for JS
  Reference *ComponentParam // When exposing the parameter to JS, it refers to a server parameter
}

type ComponentConfig struct {
  Params map[string]ComponentParam
}

type ComponentFunc func(scope *Scope)

// JavascriptParam details of a JS parameter of this component
type JavascriptParam struct {
  Name        string
  Description string
  Type        string
  Required    bool
}

// AssetJavascript Representa um recurso necessário por um componente
type Javascript struct {
  Code   string
  Params []JavascriptParam
}

// Component a referencia para um componente
type Component struct {
}

/*
CreateComponent cria um componente. Um component é uma estrutura reutilizável que possui características especiais
*/
func CreateComponent(node *Node, attrs *Attributes, t *Compiler) {

}
