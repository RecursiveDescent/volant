package parser

type Parser struct {
	Lexer     *Lexer
	tokens    []Token
	position  int
	forkedPos int
}

func (parser *Parser) ReadToken() Token {
	for parser.position >= len(parser.tokens) {
		parser.tokens = append(parser.tokens, parser.Lexer.NextToken())
	}
	return parser.tokens[parser.position]
}

func (parser *Parser) eatLastToken() {
	parser.position++
}

func (parser *Parser) fork() {
	parser.forkedPos = parser.position
}

func (parser *Parser) moveToFork() {
	parser.position = parser.forkedPos
}

func (parser *Parser) expect(primary PrimaryTokenType, secondary SecondaryTokenType) Token {
	token := parser.ReadToken()

	if primary == PrimaryNullType {
		if token.SecondaryType != secondary {
			NewError(SyntaxError, "expected "+SecondaryTypes[secondary]+", got "+token.Serialize(), token.Line, token.Column)
		}
	} else if secondary == SecondaryNullType {
		if token.PrimaryType != primary {
			NewError(SyntaxError, "expected "+PrimaryTypes[primary]+", got "+token.Serialize(), token.Line, token.Column)
		}
	} else {
		if token.PrimaryType != primary || token.SecondaryType != secondary {
			// Error: expected {primary, secondary}, got {token}
			NewError(SyntaxError, "expected "+PrimaryTypes[primary]+" and "+SecondaryTypes[secondary]+", got "+token.Serialize(), token.Line, token.Column)
		}
	}

	return token
}

func (parser *Parser) ParseGlobalStatement() Statement {
	var statement Statement

	if token := parser.ReadToken(); token.PrimaryType == ImportKeyword {
		parser.eatLastToken()
		statement = parser.parseImport()
	} else if token.PrimaryType == StructKeyword {
		parser.eatLastToken()
		statement = parser.parseStructTypedef()
	} else if token.PrimaryType == EnumKeyword {
		parser.eatLastToken()
		statement = parser.parseEnumTypedef()
	} else if token.PrimaryType == TupleKeyword {
		parser.eatLastToken()
		statement = parser.parseTupleTypedef()
	} else if token.PrimaryType == Identifier {
		statement = parser.parseDeclaration()
	} else {
		// Error: Invalid token {token}
	}

	parser.expect(SemiColon, SecondaryNullType)
	parser.eatLastToken()

	return statement
}

func (parser *Parser) parseStructTypedef() Statement {
	return parser.parseStructType(false)
}

func (parser *Parser) parseTupleTypedef() Statement {
	return parser.parseTupleType(false)
}

func (parser *Parser) parseEnumTypedef() Statement {
	return parser.parseEnumType()
}

func (parser *Parser) parseImport() Import {
	var imprt Import

	if token := parser.ReadToken(); token.PrimaryType == LeftParen {
		parser.eatLastToken()

		for token2 := parser.ReadToken(); token2.PrimaryType == Comma; parser.eatLastToken() {
			imprt.Paths = append(imprt.Paths, parser.expect(StringLiteral, SecondaryNullType))
			parser.eatLastToken()
		}

		parser.expect(RightParen, SecondaryNullType)
		parser.eatLastToken()

	} else if token.PrimaryType == StringLiteral {
		imprt.Paths = append(imprt.Paths, token)
		parser.eatLastToken()
	} else {
		// Error: expected string literal, got {token]
	}

	return imprt
}

func (parser *Parser) parseFunctionExpression() FunctionExpression {
	function := FunctionExpression{}
	function.Type = FunctionType(0)

	// check for async/work/inline keyword
	if token := parser.ReadToken(); token.PrimaryType == InlineKeyword {
		function.Type = function.Type | InlineFunction
		parser.eatLastToken()
	}

	if token := parser.ReadToken(); token.PrimaryType == AsyncKeyword {
		function.Type = function.Type | AsyncFunction
		parser.eatLastToken()
	} else if token.PrimaryType == WorkKeyword {
		function.Type = function.Type | WorkFunction
		parser.eatLastToken()
	} else {
		function.Type = function.Type | OrdFunction
	}

	if token := parser.ReadToken(); token.PrimaryType == InlineKeyword {
		function.Type = function.Type | InlineFunction
		parser.eatLastToken()
	}

	// parse arguments
	parser.expect(LeftParen, SecondaryNullType)
	parser.eatLastToken()

	function.Args = parser.parseFunctionArgs()

	parser.expect(RightParen, SecondaryNullType)
	parser.eatLastToken()

	// parse return types
	if token := parser.ReadToken(); token.PrimaryType == LeftParen {
		parser.eatLastToken()
		function.ReturnTypes = parser.parseTypeArray()

		parser.expect(RightParen, SecondaryNullType)
		parser.eatLastToken()
	} else if token.PrimaryType != LeftCurlyBrace {
		function.ReturnTypes = []TypeStruct{parser.parseType(false, false)}
	}

	// parse code block
	function.Block = parser.parseBlock()
	return function
}

func (parser *Parser) parseFunctionArgs() []ArgStruct {
	args := []ArgStruct{}

	if token := parser.ReadToken(); token.PrimaryType == RightParen {
		return args
	}

	args = append(args, parser.parseFunctionArg())

	for token := parser.ReadToken(); token.PrimaryType == Comma; parser.eatLastToken() {
		args = append(args, parser.parseFunctionArg())
	}
	return args
}

func (parser *Parser) parseFunctionArg() ArgStruct {
	arg := ArgStruct{}

	arg.Identifier = parser.expect(Identifier, SecondaryNullType)
	parser.eatLastToken()

	parser.expect(PrimaryNullType, Colon)
	parser.eatLastToken()

	arg.Type = parser.parseType(false, false)
	return arg
}

func (parser *Parser) parseType(allowTypeDefs bool, alllowUnnamed bool) TypeStruct {
	var pointerIndex byte = 0
	var typ TypeStruct

	for token := parser.ReadToken(); token.SecondaryType == Mul; token = parser.ReadToken() {
		pointerIndex++
		parser.eatLastToken()
	}

	if token := parser.ReadToken(); token.PrimaryType == StructKeyword {
		if allowTypeDefs {
			parser.eatLastToken()
			structType := parser.parseStructType(alllowUnnamed)
			typ.Type = StructType
			typ.StructType = structType
		} else {
			// Error: type not allowed
		}
	} else if token.PrimaryType == TupleKeyword {
		if allowTypeDefs {
			parser.eatLastToken()
			tupleType := parser.parseTupleType(alllowUnnamed)
			typ.Type = TupleType
			typ.TupleType = tupleType
		} else {
			// Error: type not allowed
		}
	} else if token.PrimaryType == FunctionKeyword {
		parser.eatLastToken()
		funcType := parser.parseFunctionType()
		typ.Type = FuncType
		typ.FuncType = funcType
	} else if token.PrimaryType == Identifier {
		parser.eatLastToken()
		typ.Identifier = token
		typ.Type = IdentifierType
	}

	typ.PointerIndex = pointerIndex
	return typ
}

func (parser *Parser) parseTypeArray() []TypeStruct {
	types := []TypeStruct{}

	if token := parser.ReadToken(); token.PrimaryType == RightParen {
		return types
	}

	types = append(types, parser.parseType(false, false))

	for token := parser.ReadToken(); token.PrimaryType == Comma; token = parser.ReadToken() {
		parser.eatLastToken()
		types = append(types, parser.parseType(false, false))
	}

	return types
}

func (parser *Parser) parseFunctionType() FunctionTypeStruct {
	function := FunctionTypeStruct{}
	function.Type = FunctionType(0)

	// check for async/work/inline keyword
	if token := parser.ReadToken(); token.PrimaryType == InlineKeyword {
		function.Type = function.Type | InlineFunction
		parser.eatLastToken()
	}

	if token := parser.ReadToken(); token.PrimaryType == AsyncKeyword {
		function.Type = function.Type | AsyncFunction
		parser.eatLastToken()
	} else if token.PrimaryType == WorkKeyword {
		function.Type = function.Type | WorkFunction
		parser.eatLastToken()
	} else {
		function.Type = OrdFunction
	}

	if token := parser.ReadToken(); token.PrimaryType == InlineKeyword {
		function.Type = function.Type | InlineFunction
		parser.eatLastToken()
	}

	// parse arguments
	parser.expect(LeftParen, SecondaryNullType)
	parser.eatLastToken()

	function.Args = parser.parseTypeArray()

	parser.expect(RightParen, SecondaryNullType)
	parser.eatLastToken()

	// parse return types
	if token := parser.ReadToken(); token.PrimaryType == LeftParen {
		parser.eatLastToken()

		function.ReturnTypes = parser.parseTypeArray()

		parser.expect(RightParen, SecondaryNullType)
		parser.eatLastToken()
	} else if token.PrimaryType != Comma && token.PrimaryType != SemiColon && token.SecondaryType != Equal && token.PrimaryType != RightParen {
		function.ReturnTypes = []TypeStruct{parser.parseType(false, false)}
	}

	return function
}

func (parser *Parser) parseStructType(allowUnnamed bool) Struct {
	strct := Struct{}

	if token := parser.ReadToken(); token.PrimaryType == Identifier {
		parser.eatLastToken()
		strct.Identifier = token
	} else if !allowUnnamed {
		// Error: expected identifier, got {token}
	}

	parser.expect(LeftCurlyBrace, SecondaryNullType)
	parser.eatLastToken()

	strct.Props = parser.parseStructProps()

	parser.expect(RightCurlyBrace, SecondaryNullType)
	parser.eatLastToken()

	return strct
}

func (parser *Parser) parseTupleType(allowUnnamed bool) Tuple {
	tupl := Tuple{}

	if token := parser.ReadToken(); token.PrimaryType == Identifier {
		parser.eatLastToken()
		tupl.Identifier = token
	} else if !allowUnnamed {
		// Error: expected identifier, got {token}
	}

	parser.expect(LeftCurlyBrace, SecondaryNullType)
	parser.eatLastToken()

	tupl.Types = parser.parseTypeArray()

	parser.expect(RightCurlyBrace, SecondaryNullType)
	parser.eatLastToken()

	return tupl
}

func (parser *Parser) parseEnumType() Enum {
	enum := Enum{}

	enum.Name = parser.expect(Identifier, SecondaryNullType)
	parser.eatLastToken()

	parser.expect(LeftCurlyBrace, SecondaryNullType)
	parser.eatLastToken()

	for parser.ReadToken().PrimaryType == Comma {
		parser.eatLastToken()

		token := parser.expect(Identifier, SecondaryNullType)
		parser.eatLastToken()

		enum.Identifiers = append(enum.Identifiers, token)

		if token2 := parser.ReadToken(); token2.SecondaryType == Equal {
			parser.eatLastToken()
			enum.Values = append(enum.Values, parser.parseExpression())
		} /* else {
			enum.Values = append(enum.Values)
		} */
	}

	parser.expect(RightCurlyBrace, SecondaryNullType)
	return enum
}

func (parser *Parser) parseStructProps() []StructPropStruct {
	props := []StructPropStruct{}
	props = append(props, parser.parseStructProp())

	for token := parser.ReadToken(); token.PrimaryType == SemiColon; parser.eatLastToken() {
		props = append(props, parser.parseStructProp())
	}
	return props
}

func (parser *Parser) parseStructProp() StructPropStruct {
	prop := StructPropStruct{}

	if token := parser.ReadToken(); token.SecondaryType == DotDot {
		parser.eatLastToken()

		if next := parser.ReadToken(); next.PrimaryType == Identifier {
			prop.Identifier = next
			return prop
		}
	} else if token.PrimaryType == Identifier {
		prop.Identifier = token
		prop.Type = parser.parseType(true, true)

		if prop.Type.Type != StructType && prop.Type.Type != TupleType && parser.ReadToken().SecondaryType == Equal {
			parser.eatLastToken()
			prop.Value = parser.parseExpression()
		}
	} else {
		// Error: expected identifier, got {token}
	}

	return prop
}

func (parser *Parser) parseExpressionArray() []Expression {
	exprs := []Expression{}
	exprs = append(exprs, parser.parseExpression())

	for token := parser.ReadToken(); token.PrimaryType == Comma; token = parser.ReadToken() {
		parser.eatLastToken()
		exprs = append(exprs, parser.parseExpression())
	}

	return exprs
}

// var1, var2, ...varn :[type1, type2, ...typen][= val1, val2, ...valn]
func (parser *Parser) parseDeclaration() Declaration {
	declaration := Declaration{}

	declaration.Identifiers = append(declaration.Identifiers, parser.expect(Identifier, SecondaryNullType))
	parser.eatLastToken()

	for parser.ReadToken().PrimaryType == Comma {
		parser.eatLastToken()

		declaration.Identifiers = append(declaration.Identifiers, parser.expect(Identifier, SecondaryNullType))
		parser.eatLastToken()
	}

	parser.expect(PrimaryNullType, Colon)
	parser.eatLastToken()

	if next := parser.ReadToken(); next.SecondaryType != Equal {
		declaration.Types = parser.parseTypeArray()
	} else {
		declaration.Types = []TypeStruct{}
	}

	if next := parser.ReadToken(); next.SecondaryType == Equal {
		parser.eatLastToken()
		declaration.Values = parser.parseExpressionArray()
	}

	return declaration
}

func (parser *Parser) parseIfElse() IfElseBlock {
	ifelseblock := IfElseBlock{}

	statement := parser.parseStatement()

	if parser.ReadToken().PrimaryType == SemiColon {
		parser.eatLastToken()
		ifelseblock.InitStatement = statement
		ifelseblock.HasInitStmt = true
		ifelseblock.Conditions = append(ifelseblock.Conditions, parser.parseExpression())
	} else {
		switch statement.(type) {
		case Expression:
			ifelseblock.Conditions = append(ifelseblock.Conditions, statement.(Expression))
		default:
			// Error: expected an expression, got {statement}
		}
	}

	ifelseblock.Blocks = append(ifelseblock.Blocks, parser.parseBlock())

	for token := parser.ReadToken(); token.PrimaryType == ElseKeyword; token = parser.ReadToken() {
		parser.eatLastToken()
		if next := parser.ReadToken(); next.PrimaryType == IfKeyword {
			parser.eatLastToken()
			ifelseblock.Conditions = append(ifelseblock.Conditions, parser.parseExpression())
			ifelseblock.Blocks = append(ifelseblock.Blocks, parser.parseBlock())
		} else {
			ifelseblock.ElseBlock = parser.parseBlock()
		}
	}

	return ifelseblock
}

func (parser *Parser) parseLoop() Loop {
	loop := Loop{}

	if parser.ReadToken().PrimaryType == LeftCurlyBrace {
		loop.Type = NoneLoop
	} else {
		statement := parser.parseStatement()

		if parser.ReadToken().PrimaryType == SemiColon {
			parser.eatLastToken()

			loop.InitStatement = statement
			loop.Condition = parser.parseExpression()

			if parser.ReadToken().PrimaryType == SemiColon {
				parser.eatLastToken()

				if parser.ReadToken().PrimaryType == LeftCurlyBrace {
					loop.Type = InitCond
				} else {
					loop.LoopStatement = parser.parseStatement()
					loop.Type = InitCondLoop
				}
			} else {
				loop.Type = InitCond
			}
		} else {
			switch statement.(type) {
			case Expression:
				loop.Type = Cond
				loop.Condition = statement.(Expression)
			default:
				// Error: expected an expression, got {statement}
			}
		}
	}

	loop.Block = parser.parseBlock()
	return loop
}

func (parser *Parser) parseSwitch() Switch {
	swtch := Switch{}

	if parser.ReadToken().PrimaryType != LeftCurlyBrace {
		statement := parser.parseStatement()

		if parser.ReadToken().PrimaryType == SemiColon {
			parser.eatLastToken()
			swtch.InitStatement = statement

			if parser.ReadToken().PrimaryType != LeftCurlyBrace {
				statement2 := parser.parseStatement()

				switch statement2.(type) {
				case Expression:
					swtch.Type = InitCondSwitch
					swtch.Condition = statement2.(Expression)
				default:
					// Error: Expected an expression, got {statement2}
				}
			}
		} else {
			switch statement.(type) {
			case Expression:
				swtch.Type = CondSwitch
				swtch.Condition = statement.(Expression)
			default:
				// Error: expected an expression, got {statement}
			}
		}
	}

	parser.expect(LeftCurlyBrace, SecondaryNullType)
	parser.eatLastToken()

	for parser.ReadToken().PrimaryType == CaseKeyword {
		parser.eatLastToken()

		Case := CaseStruct{}
		Case.Condition = parser.parseExpression()

		parser.expect(PrimaryNullType, Colon)
		parser.eatLastToken()

		for token := parser.ReadToken(); token.PrimaryType != CaseKeyword && token.PrimaryType != DefaultKeyword; token = parser.ReadToken() {

			if token.PrimaryType == SemiColon {
				parser.eatLastToken()
			} else if token.PrimaryType == RightCurlyBrace {
				swtch.Cases = append(swtch.Cases, Case)
				parser.eatLastToken()
				return swtch
			} else {
				Case.Statements = append(Case.Statements, parser.parseStatement())
			}
		}

		swtch.Cases = append(swtch.Cases, Case)
	}

	if parser.ReadToken().PrimaryType == DefaultKeyword {
		parser.eatLastToken()
		parser.expect(PrimaryNullType, Colon)
		parser.eatLastToken()

		DefaultCase := Block{}
		swtch.HasDefaultCase = true

		for token := parser.ReadToken(); token.PrimaryType != CaseKeyword; token = parser.ReadToken() {

			if token.PrimaryType == SemiColon {
				parser.eatLastToken()
			} else if token.PrimaryType == RightCurlyBrace {
				swtch.DefaultCase = DefaultCase
				parser.eatLastToken()
				return swtch
			} else {
				DefaultCase.Statements = append(DefaultCase.Statements, parser.parseStatement())
			}
		}
	}

	return swtch
}

func (parser *Parser) parseBlock() Block {
	block := Block{}

	parser.expect(LeftCurlyBrace, SecondaryNullType)
	parser.eatLastToken()

	for token := parser.ReadToken(); token.PrimaryType != RightCurlyBrace; token = parser.ReadToken() {
		block.Statements = append(block.Statements, parser.parseStatement())
		if parser.ReadToken().PrimaryType == SemiColon {
			parser.eatLastToken()
		}
	}

	parser.eatLastToken()
	return block
}

func (parser *Parser) parseReturn() Return {
	return Return{Values: parser.parseExpressionArray()}
}

func (parser *Parser) parseStatement() Statement {

	switch parser.ReadToken().PrimaryType {
	case IfKeyword:
		parser.eatLastToken()
		return parser.parseIfElse()
	case SwitchKeyword:
		parser.eatLastToken()
		return parser.parseSwitch()
	case ForKeyword:
		parser.eatLastToken()
		return parser.parseLoop()
	/*
		case DeferKeyword:
			parser.eatLastToken()
			return parser.parseDefer()
	*/
	case LeftCurlyBrace:
		st := parser.parseBlock()
		parser.expect(SemiColon, SecondaryNullType)
		parser.eatLastToken()
		return st
	case ReturnKeyword:
		parser.eatLastToken()
		st := parser.parseReturn()
		parser.expect(SemiColon, SecondaryNullType)
		parser.eatLastToken()
		return st
	default:
		parser.fork()
		expr := parser.parseExpression()

		if token := parser.ReadToken(); token.PrimaryType == AssignmentOperator {
			parser.moveToFork()
			return parser.parseAssignment()
		} else if token.PrimaryType == Comma {
			parser.moveToFork()
			return parser.parseDeclarationOrAssignment()
		} else if token.SecondaryType == Colon {
			parser.moveToFork()
			return parser.parseDeclaration()
		}

		return expr
	}
}

func (parser *Parser) parseAssignment() Assignment {
	as := Assignment{}

	as.Variables = parser.parseExpressionArray()

	parser.expect(AssignmentOperator, SecondaryNullType)
	as.Op = parser.ReadToken()
	parser.eatLastToken()

	if as.Op.SecondaryType != AddAdd && as.Op.SecondaryType != SubSub {
		as.Values = parser.parseExpressionArray()
	}
	return as
}

func (parser *Parser) parseDeclarationOrAssignment() Statement {
	parser.fork()

	for token := parser.ReadToken(); true; token = parser.ReadToken() {
		if token.PrimaryType == AssignmentOperator {
			parser.moveToFork()
			return parser.parseAssignment()
		} else if token.SecondaryType == Colon {
			parser.moveToFork()
			return parser.parseDeclaration()
		}
		parser.eatLastToken()
	}

	return Declaration{}
}

func (parser *Parser) parseFunctionCall(name Token) FunctionCall {
	parser.expect(LeftParen, SecondaryNullType)
	parser.eatLastToken()

	functionCall := FunctionCall{Name: name, Args: parser.parseExpressionArray()}

	parser.expect(RightParen, SecondaryNullType)
	parser.eatLastToken()

	return functionCall
}

func (parser *Parser) parseExpression() Expression {
	token := parser.ReadToken()

	if token.PrimaryType == FunctionKeyword {
		parser.eatLastToken()
		expr := parser.parseFunctionExpression()
		return Expression(expr)
	} else if token.PrimaryType == Identifier {
		parser.eatLastToken()
		if next := parser.ReadToken(); next.PrimaryType == LeftParen {
			return Expression(parser.parseFunctionCall(token))
		}
	} else {
		parser.eatLastToken()
	}

	return BasicLit{Typ: U8Type, Value: token}

	/*
		var tokens []Token = make([]Token)
		ParenCount, braceCount := 0, 0

		for {
			token := parser.ReadToken()

			switch token.PrimaryType {
			case SemiColon, LeftCurlyBrace, RightCurlyBrace:
				break
			case LeftParen:
				parenCount++
				tokens = append(tokens, token)
				parser.eatLastToken()
			case RightParen:
				if parenCount > 0 {
					parenCount--
					tokens = append(tokens, token)
					parser.eatLastToken()
				} else {
					break
				}
			case LeftBrace:
				braceCount++
				tokens = append(tokens, token)
				parser.eatLastToken()
			case RightBrace:
				if braceCount > 0 {
					braceCount--
					token = append(tokens, token)
					parser.eatLastToken()
				} else {
					break
				}
			case Comma:
				if parenCount == 0 && braceCount == 0 {
					break
				} else {
					token = append(tokens, token)
					parser.eatLastToken()
				}
			default:
				token = append(tokens, token)
				parser.eatLastToken()

			}
		}

		return exprHelper(tokens)
	*/
}

/*
func exprHelper(token []Token) Expression{


}*/
