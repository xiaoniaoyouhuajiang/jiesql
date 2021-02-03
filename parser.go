package jiesql

import (
	"errors"
	"fmt"
)

func tokenFromKeyword(k keyword) token {
	return token{
		kind:  keywordKind,
		value: string(k),
	}
}

func tokenFromSymbol(s symbol) token {
	return token{
		kind:  symbolKind,
		value: string(s),
	}
}

// helper_functtion: 根据tokens和cursor来判断当前扫描的token是否为预期token
func expectToken(tokens []*token, cursor uint, t token) bool {
	if cursor >= uint(len(tokens)) {
		return false
	}

	return t.equals(tokens[cursor])
}

// 出现问题时，打印出出问题的token以及方位。
func helpMessage(tokens []*token, cursor uint, msg string) {
	var c *token
	if cursor+1 < uint(len(tokens)) {
		c = tokens[cursor+1]
	} else {
		c = tokens[cursor]
	}

	fmt.Printf("[%d,%d]: %s, got: %s\n", c.loc.line, c.loc.col, msg, c.value)
}

// 解析主函数
// 主要使用parseStatement函数解析tokens
func Parse(source string) (*Ast, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}

	a := Ast{}
	cursor := uint(0)
	for cursor < uint(len(tokens)) {
		stmt, newCursor, ok := parseStatement(tokens, cursor, tokenFromSymbol(semicolonSymbol))
		if !ok {
			helpMessage(tokens, cursor, "Expected statement")
			return nil, errors.New("Failed to parse, expected statement")
		}
		cursor = newCursor

		a.Statements = append(a.Statements, stmt)

		atLeastOneSemicolon := false
		for expectToken(tokens, cursor, tokenFromSymbol(semicolonSymbol)) {
			cursor++
			atLeastOneSemicolon = true
		}

		if !atLeastOneSemicolon {
			helpMessage(tokens, cursor, "Expected semi-colon delimiter between statements")
			return nil, errors.New("Missing semi-colon between statements")
		}
	}

	return &a, nil
}

// 一共就三种命令，select, insert, create
func parseStatement(tokens []*token, initialCursor uint, delimiter token) (*Statement, uint, bool) {
	cursor := initialCursor

	semicolonToken := tokenFromSymbol(semicolonSymbol)
	slct, newCursor, ok := parseSelectStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			Kind:            SelectKind,
			SelectStatement: slct,
		}, newCursor, true
	}

	// Look for a INSERT statement
	inst, newCursor, ok := parseInsertStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			Kind:            InsertKind,
			InsertStatement: inst,
		}, newCursor, true
	}

	// Look for a CREATE statement
	crtTbl, newCursor, ok := parseCreateTableStatement(tokens, cursor, semicolonToken)
	if ok {
		return &Statement{
			Kind:                 CreateTableKind,
			CreateTableStatement: crtTbl,
		}, newCursor, true
	}

	return nil, initialCursor, false
}

/* select mode
1. SELECT
2. $expression [, ...]
3. FROM
4. $table-name
*/
// 切记辅助函数是需要返回新的 cursor来让parser（parse函数）进行定位
func parseSelectStatement(tokens []*token, initialCursor uint, delimiter token) (*SelectStatement, uint, bool) {
	var ok bool
	cursor := initialCursor
	_, cursor, ok = parseToken(tokens, cursor, tokenFromKeyword(selectKeyword))
	if !ok {
		return nil, initialCursor, false
	}

	slct := SelectStatement{}

	fromToken := tokenFromKeyword(fromKeyword)
	exps, newCursor, ok := parseExpressions(tokens, cursor, []token{tokenFromKeyword(fromKeyword), delimiter})
	if !ok {
		return nil, initialCursor, false
	}

	slct.item = *exps
	cursor = newCursor

	if expectToken(tokens, cursor, tokenFromKeyword(fromKeyword)) {
		cursor++

		from, newCursor, ok := parseTokenKind(tokens, cursor, identifierKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected FROM token")
			return nil, initialCursor, false
		}

		// 注意这里是取实体的值
		slct.from = *from
		cursor = newCursor
	}

	return &slct, cursor, true
}

func parseSelectItem(tokens []*token, initialCursor uint, delimiters []token) (*[]*selectItem, uint, bool) {
	cursor := initialCursor

	s := []*selectItem{}
outer:
	for {
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}

		current := tokens[cursor]
		for _, delimiter := range delimiters {
			if delimiter.equals(current) {
				break outer
			}
		}

		if len(s) > 0 {
			if !expectToken(tokens, cursor, tokenFromSymbol(commaSymbol)) {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}

			cursor++
		}

		var si selectItem
		if expectToken(tokens, cursor, tokenFromSymbol(asteriskSymbol)) {
			si = selectItem{asterisk: true}
			cursor++
		} else {
			exp, newCursor, ok := parseExpression(tokens, cursor, tokenFromSymbol(commaSymbol))
			if !ok {
				helpMessage(tokens, cursor, "Expected expression")
				return nil, initialCursor, false
			}

			cursor = newCursor
			si.exp = exp

			if expectToken(tokens, cursor, tokenFromKeyword(asKeyword)) {
				cursor++

				id, newCursor, ok := parseToken(tokens, cursor, identifierKind)
				if !ok {
					helpMessage(tokens, cursor, "Expected identifier after AS")
					return nil, initialCursor, false
				}

				cursor = newCursor
				si.as = id
			}
		}

		s = append(s, &si)
	}

	return &s, cursor, true
}

// 解析一个token
// 注意最早版本该函数的第三个参数是tokenKind类型，在版本二中重构了部分的代码
// 版本二中的末尾参数直接就是一个待确认的token，引来了一大波改动
func parseToken(tokens []*token, initialCursor uint, t token) (*token, uint, bool) {
	cursor := initialCursor

	// 如果当前游标超出了初始游标的——代表有问题
	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}

	current := tokens[cursor]
	if t.equals(current) {
		return current, cursor + 1, true
	}

	return nil, initialCursor, false
}

// 如果参数中的tokenKind和调用时碰到的一样，就会consume掉当前token
func parseTokenKind(tokens []*token, initialCursor uint, kind tokenKind) (*token, uint, bool) {
	cursor := initialCursor

	if cursor >= uint(len(tokens)) {
		return nil, initialCursor, false
	}

	current := tokens[cursor]
	if current.kind == kind {
		return current, cursor + 1, true
	}

	return nil, initialCursor, false
}

// parseExpressions的第三个参数是[]token, 用来装分隔符的一个容器，遇到任意其中分隔符都要break
// 初值部分用了parseExpression这个接口，注意这个不是递归调用
func parseExpressions(tokens []*token, initialCursor uint, delimiter token) (*[]*expression, uint, bool) {
	cursor := initialCursor

	exps := []*expression{}

	for {
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}

		// Look for delimiter
		current := tokens[cursor]
		if delimiter.equals(current) {
			break
		}

		// 对逗号的处理(这一步已经交由parseToken来处理)
		if len(exps) > 0 {
			var ok bool
			_, cursor, ok = parseToken(tokens, cursor, tokenFromSymbol(commaSymbol))
			if !ok {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}
		}

		// Look for expression
		exp, newCursor, ok := parseExpression(tokens, cursor, []token{tokenFromSymbol(commaSymbol), tokenFromSymbol(rightParenSymbol)}, 0)
		if !ok {
			helpMessage(tokens, cursor, "Expected expression")
			return nil, initialCursor, false
		}
		cursor = newCursor

		exps = append(exps, exp)
	}

	return &exps, cursor, true
}

/*
原版parseExpression
func parseExpression(tokens []*token, initialCursor uint, _ token) (*expression, uint, bool) {
	cursor := initialCursor

	kinds := []tokenKind{identifierKind, numericKind, stringKind}
	for _, kind := range kinds {
		t, newCursor, ok := parseToken(tokens, cursor, kind)
		if ok {
			return &expression{
				literal: t,
				kind:    literalKind,
			}, newCursor, true
		}
	}

	return nil, initialCursor, false
}
*/

// 新版，支持中缀运算符，继而使用Pratt parsing
func parseExpression(tokens []*token, initialCursor uint, delimiters []token, minBp uint) (*expression, uint, bool) {
	cursor := initialCursor

	var exp *expression
	_, newCursor, ok := parseToken(tokens, cursor, tokenFromSymbol(leftParenSymbol))
	if ok {
		cursor = newCursor
		rightParenToken := tokenFromSymbol(rightParenSymbol)

		exp, cursor, ok = parseExpression(tokens, cursor, append(delimiters, rightParenToken), minBp)
		if !ok {
			helpMessage(tokens, cursor, "Expected expression after opening paren")
			return nil, initialCursor, false
		}

		_, cursor, ok = parseToken(tokens, cursor, rightParenToken)
		if !ok {
			helpMessage(tokens, cursor, "Expected closing paren")
			return nil, initialCursor, false
		}
	} else {
		exp, cursor, ok = parseLiteralExpression(tokens, cursor)
		if !ok {
			return nil, initialCursor, false
		}
	}

	//
	lastCursor := cursor
outer:
	for cursor < uint(len(tokens)) {
		for _, d := range delimiters {
			_, _, ok = parseToken(tokens, cursor, d)
			if ok {
				break outer
			}
		}

		// binary operation的token列表
		binOps := []token{
			tokenFromKeyword(andKeyword),
			tokenFromKeyword(orKeyword),
			tokenFromKeyword(eqSymbol),
			tokenFromKeyword(neqSymbol),
			tokenFromKeyword(concatSymbol),
			tokenFromKeyword(plusSymbol),
		}

		var op *token = nil
		for _, bo := range binOps {
			var t *token
			t, cursor, ok = parseToken(tokens, cursor, bo)
			if ok {
				op = t
				break
			}
		}

		if op == nil {
			helpMessage(tokens, cursor, "Expected binary operation")
			return nil, initialCursor, false
		}

		bp := op.bindingPower()
		if bp < minBp {
			cursor = lastCursor
			break
		}

		b, newCursor, ok := parseExpression(tokens, cursor, delimiters, bp)
		if !ok {
			helpMessage(tokens, cursor, "Expected right operand")
			return nil, initialCursor, false
		}
		exp = &expression{
			binary: &binaryExpression{
				*exp,
				*b,
				*op,
			},
			kind: binaryKind,
		}
		cursor = newCursor
		lastCursor = cursor
	}

	return exp, cursor, true
}

func (t token) bindingPower() uint {
	switch t.kind {
	case keywordKind:
		switch keyword(t.value) {
		case andKeyword:
			fallthrough
		case orKeyword:
			return 1
		}
	case symbolKind:
		switch symbol(t.value) {
		case eqSymbol:
			fallthrough
		case neqSymbol:
			fallthrough
		case concatSymbol:
			fallthrough
		case plusSymbol:
			return 3
		}
	}

	return 0
}

/*
Insert mode
1. INSERT
2. INTO
3. $table-name
4. VALUES
5. (
6. $expression [, ...]
7. )
*/
func parseInsertStatement(tokens []*token, initialCursor uint, delimiter token) (*InsertStatement, uint, bool) {
	cursor := initialCursor

	// Look for INSERT
	if !expectToken(tokens, cursor, tokenFromKeyword(insertKeyword)) {
		return nil, initialCursor, false
	}
	cursor++

	// Look for INTO
	if !expectToken(tokens, cursor, tokenFromKeyword(intoKeyword)) {
		helpMessage(tokens, cursor, "Expected into")
		return nil, initialCursor, false
	}
	cursor++

	// Look for table name
	table, newCursor, ok := parseToken(tokens, cursor, identifierKind)
	if !ok {
		helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	// Look for VALUES
	if !expectToken(tokens, cursor, tokenFromKeyword(valuesKeyword)) {
		helpMessage(tokens, cursor, "Expected VALUES")
		return nil, initialCursor, false
	}
	cursor++

	// Look for left paren
	if !expectToken(tokens, cursor, tokenFromSymbol(leftParenSymbol)) {
		helpMessage(tokens, cursor, "Expected left paren")
		return nil, initialCursor, false
	}
	cursor++

	// Look for expression list
	values, newCursor, ok := parseExpressions(tokens, cursor, []token{tokenFromSymbol(rightParenSymbol)})
	if !ok {
		return nil, initialCursor, false
	}
	cursor = newCursor

	// Look for right paren
	if !expectToken(tokens, cursor, tokenFromSymbol(rightParenSymbol)) {
		helpMessage(tokens, cursor, "Expected right paren")
		return nil, initialCursor, false
	}
	cursor++

	return &InsertStatement{
		table:  *table,
		values: values,
	}, cursor, true
}

/*
Create mode
1. CREATE
2. $table-name
3. (
4. [$column-name $column-type [, ...]]
5. )
*/
func parseCreateTableStatement(tokens []*token, initialCursor uint, delimiter token) (*CreateTableStatement, uint, bool) {
	cursor := initialCursor

	if !expectToken(tokens, cursor, tokenFromKeyword(createKeyword)) {
		return nil, initialCursor, false
	}
	cursor++

	if !expectToken(tokens, cursor, tokenFromKeyword(tableKeyword)) {
		return nil, initialCursor, false
	}
	cursor++

	name, newCursor, ok := parseToken(tokens, cursor, identifierKind)
	if !ok {
		helpMessage(tokens, cursor, "Expected table name")
		return nil, initialCursor, false
	}
	cursor = newCursor

	if !expectToken(tokens, cursor, tokenFromSymbol(leftParenSymbol)) {
		helpMessage(tokens, cursor, "Expected left parenthesis")
		return nil, initialCursor, false
	}
	cursor++

	cols, newCursor, ok := parseColumnDefinitions(tokens, cursor, tokenFromSymbol(rightParenSymbol))
	if !ok {
		return nil, initialCursor, false
	}
	cursor = newCursor

	if !expectToken(tokens, cursor, tokenFromSymbol(rightParenSymbol)) {
		helpMessage(tokens, cursor, "Expected right parenthesis")
		return nil, initialCursor, false
	}
	cursor++

	return &CreateTableStatement{
		name: *name,
		cols: cols,
	}, cursor, true
}

func parseColumnDefinitions(tokens []*token, initialCursor uint, delimiter token) (*[]*columnDefinition, uint, bool) {
	cursor := initialCursor

	cds := []*columnDefinition{}
	for {
		if cursor >= uint(len(tokens)) {
			return nil, initialCursor, false
		}

		// Look for a delimiter
		current := tokens[cursor]
		if delimiter.equals(current) {
			break
		}

		// Look for a comma
		if len(cds) > 0 {
			if !expectToken(tokens, cursor, tokenFromSymbol(commaSymbol)) {
				helpMessage(tokens, cursor, "Expected comma")
				return nil, initialCursor, false
			}

			cursor++
		}

		// Look for a column name
		id, newCursor, ok := parseToken(tokens, cursor, identifierKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected column name")
			return nil, initialCursor, false
		}
		cursor = newCursor

		// Look for a column type
		ty, newCursor, ok := parseToken(tokens, cursor, keywordKind)
		if !ok {
			helpMessage(tokens, cursor, "Expected column type")
			return nil, initialCursor, false
		}
		cursor = newCursor

		cds = append(cds, &columnDefinition{
			name:     *id,
			datatype: *ty,
		})
	}

	return &cds, cursor, true
}
