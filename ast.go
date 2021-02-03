package jiesql

type Ast struct {
	Statements []*Statement
}

type AstKind uint

const (
	literalKind expressionKind = iota
	binaryKind
)

type binaryExpression struct {
	a  expression
	b  expression
	op token
}

const (
	SelectKind AstKind = iota
	CreateTableKind
	InsertKind
)

type Statement struct {
	SelectStatement      *SelectStatement
	CreateTableStatement *CreateTableStatement
	InsertStatement      *InsertStatement
	Kind                 AstKind
}

type InsertStatement struct {
	table  token
	values *[]*expression
}

type expressionKind uint

type expression struct {
	literal *token
	binary  *binaryExpression
	kind    expressionKind
}

type columnDefinition struct {
	name     token
	datatype token
}

type CreateTableStatement struct {
	name token
	cols *[]*columnDefinition
}

type SelectStatement struct {
	item *[]*expression
	from *fromItem
}

type selectItem struct {
	exp      *expression
	asterisk bool
	as       *token
}

type fromItem struct {
	table *token
}
