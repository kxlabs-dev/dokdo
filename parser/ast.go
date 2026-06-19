package parser

type QueryFile struct {
	Namespace string
	Queries   []QueryNode
}

type QueryNode struct {
	Name     string
	ParamRef string
	Body     []SQLNode
}

type SQLNode interface{ sqlNode() }

type SQLText struct {
	Text string
}

type BindParam struct {
	Path string // name / user.score
}

type RawParam struct {
	Path string
}

type WhereNode struct {
	Children []SQLNode
}

type IfNode struct {
	Cond    string
	Then    []SQLNode
	ElseIfs []ElseIfClause
	Else    []SQLNode
}

type ElseIfClause struct {
	Cond string
	Body []SQLNode
}

type SwitchNode struct {
	Expr    string
	Cases   []CaseClause
	Default []SQLNode
}

type CaseClause struct {
	Value string
	Body  []SQLNode
}

type ForNode struct {
	ItemVar    string
	Collection string
	Body       []SQLNode
}

func (*SQLText) sqlNode()    {}
func (*BindParam) sqlNode()  {}
func (*RawParam) sqlNode()   {}
func (*WhereNode) sqlNode()  {}
func (*IfNode) sqlNode()     {}
func (*SwitchNode) sqlNode() {}
func (*ForNode) sqlNode()    {}
