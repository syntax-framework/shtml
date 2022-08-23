package sht

// Live interface de inter
type Live struct {
}

type ComponentFunc func(scope *Scope)

// Component a referencia para um componente
type Component struct {
}

/*
CreateComponent cria um componente. Um component é uma estrutura reutilizável que possui características especiais
*/
func CreateComponent(node *Node, attrs *Attributes, t *Compiler) {

}
