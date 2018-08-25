package ast

// Scope expands on freeVariables to track whatever the identifier is pointing to
type Scope map[Identifier]Node

func (n *NodeBase) SetScope(scope Scope) {
	n.scope = scope
}

func (n *NodeBase) Scope() Scope {
	return n.scope
}
