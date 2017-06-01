package search

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleq
	rulebody
	ruleexpression
	ruleterm
	rulefield
	rulefield_norm
	rulefield_group
	rulefield_range
	rulefield_inc_range
	rulefield_exc_range
	rulefield_name
	rulerange_value
	rulesub_q
	rulegroup
	ruleoperation
	ruleunary_op
	rulebinary_op
	ruleboolean_operator
	ruleor_operator
	ruleand_operator
	rulenot_op
	rulenot_sub_op
	rulenot_operator
	rulebang_operator
	rulerequired_op
	rulerequired_operator
	ruleprohibited_op
	ruleprohibited_operator
	ruleboost_op
	rulefuzzy_op
	rulefuzzy_param
	rulestring
	rulekeyword
	rulevalid_letter
	rulestart_letter
	ruleend_letter
	rulespecial_char
	ruleopen_paren
	ruleclose_paren
	ruleopen_incl
	ruleclose_incl
	ruleopen_excl
	ruleclose_excl
	rulespace
	rulePegText
	ruleAction0
	ruleAction1
	ruleAction2
	ruleAction3
	ruleAction4
	ruleAction5
	ruleAction6
	ruleAction7
	ruleAction8
	ruleAction9
	ruleAction10
	ruleAction11
	ruleAction12
	ruleAction13
	ruleAction14
	ruleAction15
	ruleAction16
	ruleAction17
	ruleAction18
	ruleAction19
	ruleAction20
	ruleAction21
)

var rul3s = [...]string{
	"Unknown",
	"q",
	"body",
	"expression",
	"term",
	"field",
	"field_norm",
	"field_group",
	"field_range",
	"field_inc_range",
	"field_exc_range",
	"field_name",
	"range_value",
	"sub_q",
	"group",
	"operation",
	"unary_op",
	"binary_op",
	"boolean_operator",
	"or_operator",
	"and_operator",
	"not_op",
	"not_sub_op",
	"not_operator",
	"bang_operator",
	"required_op",
	"required_operator",
	"prohibited_op",
	"prohibited_operator",
	"boost_op",
	"fuzzy_op",
	"fuzzy_param",
	"string",
	"keyword",
	"valid_letter",
	"start_letter",
	"end_letter",
	"special_char",
	"open_paren",
	"close_paren",
	"open_incl",
	"close_incl",
	"open_excl",
	"close_excl",
	"space",
	"PegText",
	"Action0",
	"Action1",
	"Action2",
	"Action3",
	"Action4",
	"Action5",
	"Action6",
	"Action7",
	"Action8",
	"Action9",
	"Action10",
	"Action11",
	"Action12",
	"Action13",
	"Action14",
	"Action15",
	"Action16",
	"Action17",
	"Action18",
	"Action19",
	"Action20",
	"Action21",
}

type token32 struct {
	pegRule
	begin, end uint32
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v", rul3s[t.pegRule], t.begin, t.end)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(pretty bool, buffer string) {
	var print func(node *node32, depth int)
	print = func(node *node32, depth int) {
		for node != nil {
			for c := 0; c < depth; c++ {
				fmt.Printf(" ")
			}
			rule := rul3s[node.pegRule]
			quote := strconv.Quote(string(([]rune(buffer)[node.begin:node.end])))
			if !pretty {
				fmt.Printf("%v %v\n", rule, quote)
			} else {
				fmt.Printf("\x1B[34m%v\x1B[m %v\n", rule, quote)
			}
			if node.up != nil {
				print(node.up, depth+1)
			}
			node = node.next
		}
	}
	print(node, 0)
}

func (node *node32) Print(buffer string) {
	node.print(false, buffer)
}

func (node *node32) PrettyPrint(buffer string) {
	node.print(true, buffer)
}

type tokens32 struct {
	tree []token32
}

func (t *tokens32) Trim(length uint32) {
	t.tree = t.tree[:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) AST() *node32 {
	type element struct {
		node *node32
		down *element
	}
	tokens := t.Tokens()
	var stack *element
	for _, token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	if stack != nil {
		return stack.node
	}
	return nil
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	t.AST().Print(buffer)
}

func (t *tokens32) PrettyPrintSyntaxTree(buffer string) {
	t.AST().PrettyPrint(buffer)
}

func (t *tokens32) Add(rule pegRule, begin, end, index uint32) {
	if tree := t.tree; int(index) >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	t.tree[index] = token32{
		pegRule: rule,
		begin:   begin,
		end:     end,
	}
}

func (t *tokens32) Tokens() []token32 {
	return t.tree
}

type Tokenizer struct {
	Token

	Buffer string
	buffer []rune
	rules  [68]func() bool
	parse  func(rule ...int) error
	reset  func()
	Pretty bool
	tokens32
}

func (p *Tokenizer) Parse(rule ...int) error {
	return p.parse(rule...)
}

func (p *Tokenizer) Reset() {
	p.reset()
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p   *Tokenizer
	max token32
}

func (e *parseError) Error() string {
	tokens, error := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return error
}

func (p *Tokenizer) PrintSyntaxTree() {
	if p.Pretty {
		p.tokens32.PrettyPrintSyntaxTree(p.Buffer)
	} else {
		p.tokens32.PrintSyntaxTree(p.Buffer)
	}
}

func (p *Tokenizer) Execute() {
	buffer, _buffer, text, begin, end := p.Buffer, p.buffer, "", 0, 0
	for _, token := range p.Tokens() {
		switch token.pegRule {

		case rulePegText:
			begin, end = int(token.begin), int(token.end)
			text = string(_buffer[begin:end])

		case ruleAction0:
			p.AddTerm(buffer[begin:end])
		case ruleAction1:
			p.StartBasic()
		case ruleAction2:
			p.StartGrouped()
		case ruleAction3:
			p.SetCompleted()
		case ruleAction4:
			p.StartRange(true)
		case ruleAction5:
			p.StartRange(false)
		case ruleAction6:
			p.AddField(buffer[begin:end])
		case ruleAction7:
			p.AddRange(buffer[begin:end])
		case ruleAction8:
			p.StartSubQuery()
		case ruleAction9:
			p.EndSubQuery()
		case ruleAction10:
			p.AddOp(OpBinOr)
		case ruleAction11:
			p.AddOp(OpBinAnd)
		case ruleAction12:
			p.StartSubQuery()
		case ruleAction13:
			p.EndSubQuery()
		case ruleAction14:
			p.AddTermOp(OpUnaryNot)
		case ruleAction15:
			p.AddTermOp(OpUnaryNot)
		case ruleAction16:
			p.AddTermOp(OpUnaryReq)
		case ruleAction17:
			p.AddTermOp(OpUnaryPro)
		case ruleAction18:
			p.AddOp(OpBoost)
		case ruleAction19:
			p.AddOp(OpFuzzy)
		case ruleAction20:
			p.AddTerm(buffer[begin:end])
		case ruleAction21:
			p.AddTerm(buffer[begin:end])

		}
	}
	_, _, _, _, _ = buffer, _buffer, text, begin, end
}

func (p *Tokenizer) Init() {
	var (
		max                  token32
		position, tokenIndex uint32
		buffer               []rune
	)
	p.reset = func() {
		max = token32{}
		position, tokenIndex = 0, 0

		p.buffer = []rune(p.Buffer)
		if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
			p.buffer = append(p.buffer, endSymbol)
		}
		buffer = p.buffer
	}
	p.reset()

	_rules := p.rules
	tree := tokens32{tree: make([]token32, math.MaxInt16)}
	p.parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokens32 = tree
		if matches {
			p.Trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	add := func(rule pegRule, begin uint32) {
		tree.Add(rule, begin, position, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 q <- <(body !.)> */
		func() bool {
			position0, tokenIndex0 := position, tokenIndex
			{
				position1 := position
				if !_rules[rulebody]() {
					goto l0
				}
				{
					position2, tokenIndex2 := position, tokenIndex
					if !matchDot() {
						goto l2
					}
					goto l0
				l2:
					position, tokenIndex = position2, tokenIndex2
				}
				add(ruleq, position1)
			}
			return true
		l0:
			position, tokenIndex = position0, tokenIndex0
			return false
		},
		/* 1 body <- <(expression / space)*> */
		func() bool {
			{
				position4 := position
			l5:
				{
					position6, tokenIndex6 := position, tokenIndex
					{
						position7, tokenIndex7 := position, tokenIndex
						{
							position9 := position
							{
								position10, tokenIndex10 := position, tokenIndex
								{
									position12 := position
									{
										position13, tokenIndex13 := position, tokenIndex
										{
											position15 := position
											{
												position16, tokenIndex16 := position, tokenIndex
												{
													position18 := position
													{
														position19, tokenIndex19 := position, tokenIndex
														if !_rules[rulenot_operator]() {
															goto l20
														}
														if !_rules[rulespace]() {
															goto l20
														}
														{
															position21, tokenIndex21 := position, tokenIndex
															if !_rules[rulefield]() {
																goto l22
															}
															goto l21
														l22:
															position, tokenIndex = position21, tokenIndex21
															if !_rules[rulefield_range]() {
																goto l23
															}
															goto l21
														l23:
															position, tokenIndex = position21, tokenIndex21
															if !_rules[ruleterm]() {
																goto l24
															}
															goto l21
														l24:
															position, tokenIndex = position21, tokenIndex21
															if !_rules[rulestring]() {
																goto l20
															}
														}
													l21:
														goto l19
													l20:
														position, tokenIndex = position19, tokenIndex19
														{
															position25 := position
															if buffer[position] != rune('!') {
																goto l17
															}
															position++
															{
																add(ruleAction15, position)
															}
															add(rulebang_operator, position25)
														}
														{
															position27, tokenIndex27 := position, tokenIndex
															if !_rules[rulespace]() {
																goto l27
															}
															goto l28
														l27:
															position, tokenIndex = position27, tokenIndex27
														}
													l28:
														{
															position29, tokenIndex29 := position, tokenIndex
															if !_rules[rulefield]() {
																goto l30
															}
															goto l29
														l30:
															position, tokenIndex = position29, tokenIndex29
															if !_rules[rulefield_range]() {
																goto l31
															}
															goto l29
														l31:
															position, tokenIndex = position29, tokenIndex29
															if !_rules[ruleterm]() {
																goto l32
															}
															goto l29
														l32:
															position, tokenIndex = position29, tokenIndex29
															if !_rules[rulestring]() {
																goto l17
															}
														}
													l29:
													}
												l19:
													add(rulenot_op, position18)
												}
												goto l16
											l17:
												position, tokenIndex = position16, tokenIndex16
												{
													switch buffer[position] {
													case '-':
														{
															position34 := position
															{
																position35, tokenIndex35 := position, tokenIndex
																if !_rules[rulevalid_letter]() {
																	goto l35
																}
																goto l14
															l35:
																position, tokenIndex = position35, tokenIndex35
															}
															{
																position36 := position
																if buffer[position] != rune('-') {
																	goto l14
																}
																position++
																{
																	add(ruleAction17, position)
																}
																add(ruleprohibited_operator, position36)
															}
															{
																position38, tokenIndex38 := position, tokenIndex
																if !_rules[rulefield]() {
																	goto l39
																}
																goto l38
															l39:
																position, tokenIndex = position38, tokenIndex38
																if !_rules[rulefield_range]() {
																	goto l40
																}
																goto l38
															l40:
																position, tokenIndex = position38, tokenIndex38
																if !_rules[ruleterm]() {
																	goto l41
																}
																goto l38
															l41:
																position, tokenIndex = position38, tokenIndex38
																if !_rules[rulestring]() {
																	goto l14
																}
															}
														l38:
															add(ruleprohibited_op, position34)
														}
														break
													case '+':
														{
															position42 := position
															{
																position43, tokenIndex43 := position, tokenIndex
																{
																	position45, tokenIndex45 := position, tokenIndex
																	if !_rules[rulevalid_letter]() {
																		goto l45
																	}
																	goto l44
																l45:
																	position, tokenIndex = position45, tokenIndex45
																}
																if !_rules[rulerequired_operator]() {
																	goto l44
																}
																{
																	position46, tokenIndex46 := position, tokenIndex
																	if !_rules[ruleterm]() {
																		goto l47
																	}
																	goto l46
																l47:
																	position, tokenIndex = position46, tokenIndex46
																	if !_rules[rulestring]() {
																		goto l44
																	}
																}
															l46:
																goto l43
															l44:
																position, tokenIndex = position43, tokenIndex43
																if !_rules[rulerequired_operator]() {
																	goto l14
																}
																{
																	position48, tokenIndex48 := position, tokenIndex
																	if !_rules[ruleterm]() {
																		goto l49
																	}
																	goto l48
																l49:
																	position, tokenIndex = position48, tokenIndex48
																	if !_rules[rulestring]() {
																		goto l14
																	}
																}
															l48:
															}
														l43:
															add(rulerequired_op, position42)
														}
														break
													default:
														{
															position50 := position
															{
																add(ruleAction12, position)
															}
															if !_rules[rulenot_operator]() {
																goto l14
															}
															if !_rules[rulespace]() {
																goto l14
															}
															if !_rules[rulesub_q]() {
																goto l14
															}
															{
																add(ruleAction13, position)
															}
															add(rulenot_sub_op, position50)
														}
														break
													}
												}

											}
										l16:
											add(ruleunary_op, position15)
										}
										goto l13
									l14:
										position, tokenIndex = position13, tokenIndex13
										{
											position54 := position
											{
												position55, tokenIndex55 := position, tokenIndex
												if !_rules[rulegroup]() {
													goto l56
												}
												goto l55
											l56:
												position, tokenIndex = position55, tokenIndex55
												if !_rules[rulefield]() {
													goto l57
												}
												goto l55
											l57:
												position, tokenIndex = position55, tokenIndex55
												if !_rules[rulefield_range]() {
													goto l58
												}
												goto l55
											l58:
												position, tokenIndex = position55, tokenIndex55
												if !_rules[ruleterm]() {
													goto l53
												}
											}
										l55:
											{
												position59, tokenIndex59 := position, tokenIndex
												if !_rules[rulespace]() {
													goto l59
												}
												goto l60
											l59:
												position, tokenIndex = position59, tokenIndex59
											}
										l60:
											{
												position61 := position
												{
													position62, tokenIndex62 := position, tokenIndex
													{
														position64 := position
														{
															position65, tokenIndex65 := position, tokenIndex
															if buffer[position] != rune('O') {
																goto l66
															}
															position++
															if buffer[position] != rune('R') {
																goto l66
															}
															position++
															goto l65
														l66:
															position, tokenIndex = position65, tokenIndex65
															if buffer[position] != rune('|') {
																goto l63
															}
															position++
															if buffer[position] != rune('|') {
																goto l63
															}
															position++
														}
													l65:
														add(ruleor_operator, position64)
													}
													{
														add(ruleAction10, position)
													}
													goto l62
												l63:
													position, tokenIndex = position62, tokenIndex62
													{
														position68 := position
														{
															position69, tokenIndex69 := position, tokenIndex
															if buffer[position] != rune('A') {
																goto l70
															}
															position++
															if buffer[position] != rune('N') {
																goto l70
															}
															position++
															if buffer[position] != rune('D') {
																goto l70
															}
															position++
															goto l69
														l70:
															position, tokenIndex = position69, tokenIndex69
															if buffer[position] != rune('&') {
																goto l53
															}
															position++
															if buffer[position] != rune('&') {
																goto l53
															}
															position++
														}
													l69:
														add(ruleand_operator, position68)
													}
													{
														add(ruleAction11, position)
													}
												}
											l62:
												add(ruleboolean_operator, position61)
											}
											if !_rules[rulespace]() {
												goto l53
											}
										l72:
											{
												position73, tokenIndex73 := position, tokenIndex
												if !_rules[rulespace]() {
													goto l73
												}
												goto l72
											l73:
												position, tokenIndex = position73, tokenIndex73
											}
											if !_rules[rulebody]() {
												goto l53
											}
											add(rulebinary_op, position54)
										}
										goto l13
									l53:
										position, tokenIndex = position13, tokenIndex13
										{
											position75 := position
											{
												position76, tokenIndex76 := position, tokenIndex
												if !_rules[ruleterm]() {
													goto l77
												}
												goto l76
											l77:
												position, tokenIndex = position76, tokenIndex76
												if !_rules[rulestring]() {
													goto l74
												}
											}
										l76:
											if buffer[position] != rune('~') {
												goto l74
											}
											position++
											{
												add(ruleAction19, position)
											}
											{
												position79, tokenIndex79 := position, tokenIndex
												if !_rules[rulefuzzy_param]() {
													goto l79
												}
												goto l80
											l79:
												position, tokenIndex = position79, tokenIndex79
											}
										l80:
											{
												position81, tokenIndex81 := position, tokenIndex
												if !_rules[rulespace]() {
													goto l82
												}
												goto l81
											l82:
												position, tokenIndex = position81, tokenIndex81
												{
													position83, tokenIndex83 := position, tokenIndex
													if !_rules[rulevalid_letter]() {
														goto l83
													}
													goto l74
												l83:
													position, tokenIndex = position83, tokenIndex83
												}
											}
										l81:
											add(rulefuzzy_op, position75)
										}
										goto l13
									l74:
										position, tokenIndex = position13, tokenIndex13
										{
											position84 := position
											{
												position85, tokenIndex85 := position, tokenIndex
												if !_rules[ruleterm]() {
													goto l86
												}
												goto l85
											l86:
												position, tokenIndex = position85, tokenIndex85
												if !_rules[rulestring]() {
													goto l11
												}
											}
										l85:
											if buffer[position] != rune('^') {
												goto l11
											}
											position++
											{
												add(ruleAction18, position)
											}
											if !_rules[rulefuzzy_param]() {
												goto l11
											}
											add(ruleboost_op, position84)
										}
									}
								l13:
									add(ruleoperation, position12)
								}
								goto l10
							l11:
								position, tokenIndex = position10, tokenIndex10
								if !_rules[rulefield]() {
									goto l88
								}
								goto l10
							l88:
								position, tokenIndex = position10, tokenIndex10
								if !_rules[rulefield_range]() {
									goto l89
								}
								goto l10
							l89:
								position, tokenIndex = position10, tokenIndex10
								{
									switch buffer[position] {
									case '"':
										if !_rules[rulestring]() {
											goto l8
										}
										break
									case '\t', '\n', '\r', ' ', '(':
										if !_rules[rulegroup]() {
											goto l8
										}
										break
									default:
										if !_rules[ruleterm]() {
											goto l8
										}
										break
									}
								}

							}
						l10:
							add(ruleexpression, position9)
						}
						goto l7
					l8:
						position, tokenIndex = position7, tokenIndex7
						if !_rules[rulespace]() {
							goto l6
						}
					}
				l7:
					goto l5
				l6:
					position, tokenIndex = position6, tokenIndex6
				}
				add(rulebody, position4)
			}
			return true
		},
		/* 2 expression <- <(operation / field / field_range / ((&('"') string) | (&('\t' | '\n' | '\r' | ' ' | '(') group) | (&('*' | '.' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | '\\' | '_' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') term)))> */
		nil,
		/* 3 term <- <(<((keyword valid_letter+) / (!keyword !'?' valid_letter))> Action0)> */
		func() bool {
			position92, tokenIndex92 := position, tokenIndex
			{
				position93 := position
				{
					position94 := position
					{
						position95, tokenIndex95 := position, tokenIndex
						if !_rules[rulekeyword]() {
							goto l96
						}
						if !_rules[rulevalid_letter]() {
							goto l96
						}
					l97:
						{
							position98, tokenIndex98 := position, tokenIndex
							if !_rules[rulevalid_letter]() {
								goto l98
							}
							goto l97
						l98:
							position, tokenIndex = position98, tokenIndex98
						}
						goto l95
					l96:
						position, tokenIndex = position95, tokenIndex95
						{
							position99, tokenIndex99 := position, tokenIndex
							if !_rules[rulekeyword]() {
								goto l99
							}
							goto l92
						l99:
							position, tokenIndex = position99, tokenIndex99
						}
						{
							position100, tokenIndex100 := position, tokenIndex
							if buffer[position] != rune('?') {
								goto l100
							}
							position++
							goto l92
						l100:
							position, tokenIndex = position100, tokenIndex100
						}
						if !_rules[rulevalid_letter]() {
							goto l92
						}
					}
				l95:
					add(rulePegText, position94)
				}
				{
					add(ruleAction0, position)
				}
				add(ruleterm, position93)
			}
			return true
		l92:
			position, tokenIndex = position92, tokenIndex92
			return false
		},
		/* 4 field <- <(field_norm / field_group)> */
		func() bool {
			position102, tokenIndex102 := position, tokenIndex
			{
				position103 := position
				{
					position104, tokenIndex104 := position, tokenIndex
					{
						position106 := position
						{
							add(ruleAction1, position)
						}
						if !_rules[rulefield_name]() {
							goto l105
						}
						if buffer[position] != rune(':') {
							goto l105
						}
						position++
						{
							position108, tokenIndex108 := position, tokenIndex
							if !_rules[ruleterm]() {
								goto l109
							}
							goto l108
						l109:
							position, tokenIndex = position108, tokenIndex108
							if !_rules[rulestring]() {
								goto l105
							}
						}
					l108:
						add(rulefield_norm, position106)
					}
					goto l104
				l105:
					position, tokenIndex = position104, tokenIndex104
					{
						position110 := position
						{
							add(ruleAction2, position)
						}
						if !_rules[rulefield_name]() {
							goto l102
						}
						if buffer[position] != rune(':') {
							goto l102
						}
						position++
						if !_rules[rulegroup]() {
							goto l102
						}
						{
							add(ruleAction3, position)
						}
						add(rulefield_group, position110)
					}
				}
			l104:
				add(rulefield, position103)
			}
			return true
		l102:
			position, tokenIndex = position102, tokenIndex102
			return false
		},
		/* 5 field_norm <- <(Action1 field_name ':' (term / string))> */
		nil,
		/* 6 field_group <- <(Action2 field_name ':' group Action3)> */
		nil,
		/* 7 field_range <- <(field_inc_range / field_exc_range)> */
		func() bool {
			position115, tokenIndex115 := position, tokenIndex
			{
				position116 := position
				{
					position117, tokenIndex117 := position, tokenIndex
					{
						position119 := position
						{
							add(ruleAction4, position)
						}
						if !_rules[rulefield_name]() {
							goto l118
						}
						if buffer[position] != rune(':') {
							goto l118
						}
						position++
						{
							position121 := position
							if buffer[position] != rune('[') {
								goto l118
							}
							position++
							add(ruleopen_incl, position121)
						}
						if !_rules[rulerange_value]() {
							goto l118
						}
						if buffer[position] != rune(' ') {
							goto l118
						}
						position++
						if buffer[position] != rune('T') {
							goto l118
						}
						position++
						if buffer[position] != rune('O') {
							goto l118
						}
						position++
						if buffer[position] != rune(' ') {
							goto l118
						}
						position++
						if !_rules[rulerange_value]() {
							goto l118
						}
						{
							position122 := position
							if buffer[position] != rune(']') {
								goto l118
							}
							position++
							add(ruleclose_incl, position122)
						}
						add(rulefield_inc_range, position119)
					}
					goto l117
				l118:
					position, tokenIndex = position117, tokenIndex117
					{
						position123 := position
						{
							add(ruleAction5, position)
						}
						if !_rules[rulefield_name]() {
							goto l115
						}
						if buffer[position] != rune(':') {
							goto l115
						}
						position++
						{
							position125 := position
							if buffer[position] != rune('{') {
								goto l115
							}
							position++
							add(ruleopen_excl, position125)
						}
						if !_rules[rulerange_value]() {
							goto l115
						}
						if buffer[position] != rune(' ') {
							goto l115
						}
						position++
						if buffer[position] != rune('T') {
							goto l115
						}
						position++
						if buffer[position] != rune('O') {
							goto l115
						}
						position++
						if buffer[position] != rune(' ') {
							goto l115
						}
						position++
						if !_rules[rulerange_value]() {
							goto l115
						}
						{
							position126 := position
							if buffer[position] != rune('}') {
								goto l115
							}
							position++
							add(ruleclose_excl, position126)
						}
						add(rulefield_exc_range, position123)
					}
				}
			l117:
				add(rulefield_range, position116)
			}
			return true
		l115:
			position, tokenIndex = position115, tokenIndex115
			return false
		},
		/* 8 field_inc_range <- <(Action4 field_name ':' open_incl range_value (' ' 'T' 'O' ' ') range_value close_incl)> */
		nil,
		/* 9 field_exc_range <- <(Action5 field_name ':' open_excl range_value (' ' 'T' 'O' ' ') range_value close_excl)> */
		nil,
		/* 10 field_name <- <(<(!keyword valid_letter+)> Action6)> */
		func() bool {
			position129, tokenIndex129 := position, tokenIndex
			{
				position130 := position
				{
					position131 := position
					{
						position132, tokenIndex132 := position, tokenIndex
						if !_rules[rulekeyword]() {
							goto l132
						}
						goto l129
					l132:
						position, tokenIndex = position132, tokenIndex132
					}
					if !_rules[rulevalid_letter]() {
						goto l129
					}
				l133:
					{
						position134, tokenIndex134 := position, tokenIndex
						if !_rules[rulevalid_letter]() {
							goto l134
						}
						goto l133
					l134:
						position, tokenIndex = position134, tokenIndex134
					}
					add(rulePegText, position131)
				}
				{
					add(ruleAction6, position)
				}
				add(rulefield_name, position130)
			}
			return true
		l129:
			position, tokenIndex = position129, tokenIndex129
			return false
		},
		/* 11 range_value <- <(<(valid_letter+ / '*')> Action7)> */
		func() bool {
			position136, tokenIndex136 := position, tokenIndex
			{
				position137 := position
				{
					position138 := position
					{
						position139, tokenIndex139 := position, tokenIndex
						if !_rules[rulevalid_letter]() {
							goto l140
						}
					l141:
						{
							position142, tokenIndex142 := position, tokenIndex
							if !_rules[rulevalid_letter]() {
								goto l142
							}
							goto l141
						l142:
							position, tokenIndex = position142, tokenIndex142
						}
						goto l139
					l140:
						position, tokenIndex = position139, tokenIndex139
						if buffer[position] != rune('*') {
							goto l136
						}
						position++
					}
				l139:
					add(rulePegText, position138)
				}
				{
					add(ruleAction7, position)
				}
				add(rulerange_value, position137)
			}
			return true
		l136:
			position, tokenIndex = position136, tokenIndex136
			return false
		},
		/* 12 sub_q <- <(open_paren body close_paren)> */
		func() bool {
			position144, tokenIndex144 := position, tokenIndex
			{
				position145 := position
				{
					position146 := position
					if buffer[position] != rune('(') {
						goto l144
					}
					position++
					add(ruleopen_paren, position146)
				}
				if !_rules[rulebody]() {
					goto l144
				}
				{
					position147 := position
					if buffer[position] != rune(')') {
						goto l144
					}
					position++
					add(ruleclose_paren, position147)
				}
				add(rulesub_q, position145)
			}
			return true
		l144:
			position, tokenIndex = position144, tokenIndex144
			return false
		},
		/* 13 group <- <(space? Action8 sub_q Action9 space?)> */
		func() bool {
			position148, tokenIndex148 := position, tokenIndex
			{
				position149 := position
				{
					position150, tokenIndex150 := position, tokenIndex
					if !_rules[rulespace]() {
						goto l150
					}
					goto l151
				l150:
					position, tokenIndex = position150, tokenIndex150
				}
			l151:
				{
					add(ruleAction8, position)
				}
				if !_rules[rulesub_q]() {
					goto l148
				}
				{
					add(ruleAction9, position)
				}
				{
					position154, tokenIndex154 := position, tokenIndex
					if !_rules[rulespace]() {
						goto l154
					}
					goto l155
				l154:
					position, tokenIndex = position154, tokenIndex154
				}
			l155:
				add(rulegroup, position149)
			}
			return true
		l148:
			position, tokenIndex = position148, tokenIndex148
			return false
		},
		/* 14 operation <- <(unary_op / binary_op / fuzzy_op / boost_op)> */
		nil,
		/* 15 unary_op <- <(not_op / ((&('-') prohibited_op) | (&('+') required_op) | (&('N') not_sub_op)))> */
		nil,
		/* 16 binary_op <- <((group / field / field_range / term) space? boolean_operator space+ body)> */
		nil,
		/* 17 boolean_operator <- <((or_operator Action10) / (and_operator Action11))> */
		nil,
		/* 18 or_operator <- <(('O' 'R') / ('|' '|'))> */
		nil,
		/* 19 and_operator <- <(('A' 'N' 'D') / ('&' '&'))> */
		nil,
		/* 20 not_op <- <((not_operator space (field / field_range / term / string)) / (bang_operator space? (field / field_range / term / string)))> */
		nil,
		/* 21 not_sub_op <- <(Action12 not_operator space sub_q Action13)> */
		nil,
		/* 22 not_operator <- <('N' 'O' 'T' Action14)> */
		func() bool {
			position164, tokenIndex164 := position, tokenIndex
			{
				position165 := position
				if buffer[position] != rune('N') {
					goto l164
				}
				position++
				if buffer[position] != rune('O') {
					goto l164
				}
				position++
				if buffer[position] != rune('T') {
					goto l164
				}
				position++
				{
					add(ruleAction14, position)
				}
				add(rulenot_operator, position165)
			}
			return true
		l164:
			position, tokenIndex = position164, tokenIndex164
			return false
		},
		/* 23 bang_operator <- <('!' Action15)> */
		nil,
		/* 24 required_op <- <((!valid_letter required_operator (term / string)) / (required_operator (term / string)))> */
		nil,
		/* 25 required_operator <- <('+' Action16)> */
		func() bool {
			position169, tokenIndex169 := position, tokenIndex
			{
				position170 := position
				if buffer[position] != rune('+') {
					goto l169
				}
				position++
				{
					add(ruleAction16, position)
				}
				add(rulerequired_operator, position170)
			}
			return true
		l169:
			position, tokenIndex = position169, tokenIndex169
			return false
		},
		/* 26 prohibited_op <- <(!valid_letter prohibited_operator (field / field_range / term / string))> */
		nil,
		/* 27 prohibited_operator <- <('-' Action17)> */
		nil,
		/* 28 boost_op <- <((term / string) '^' Action18 fuzzy_param)> */
		nil,
		/* 29 fuzzy_op <- <((term / string) '~' Action19 fuzzy_param? (space / !valid_letter))> */
		nil,
		/* 30 fuzzy_param <- <(<(([0-9] ('.' '?') [0-9]) / [0-9]+)> Action20)> */
		func() bool {
			position176, tokenIndex176 := position, tokenIndex
			{
				position177 := position
				{
					position178 := position
					{
						position179, tokenIndex179 := position, tokenIndex
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l180
						}
						position++
						if buffer[position] != rune('.') {
							goto l180
						}
						position++
						if buffer[position] != rune('?') {
							goto l180
						}
						position++
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l180
						}
						position++
						goto l179
					l180:
						position, tokenIndex = position179, tokenIndex179
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l176
						}
						position++
					l181:
						{
							position182, tokenIndex182 := position, tokenIndex
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l182
							}
							position++
							goto l181
						l182:
							position, tokenIndex = position182, tokenIndex182
						}
					}
				l179:
					add(rulePegText, position178)
				}
				{
					add(ruleAction20, position)
				}
				add(rulefuzzy_param, position177)
			}
			return true
		l176:
			position, tokenIndex = position176, tokenIndex176
			return false
		},
		/* 31 string <- <('"' <(term (space term)*)> '"' Action21)> */
		func() bool {
			position184, tokenIndex184 := position, tokenIndex
			{
				position185 := position
				if buffer[position] != rune('"') {
					goto l184
				}
				position++
				{
					position186 := position
					if !_rules[ruleterm]() {
						goto l184
					}
				l187:
					{
						position188, tokenIndex188 := position, tokenIndex
						if !_rules[rulespace]() {
							goto l188
						}
						if !_rules[ruleterm]() {
							goto l188
						}
						goto l187
					l188:
						position, tokenIndex = position188, tokenIndex188
					}
					add(rulePegText, position186)
				}
				if buffer[position] != rune('"') {
					goto l184
				}
				position++
				{
					add(ruleAction21, position)
				}
				add(rulestring, position185)
			}
			return true
		l184:
			position, tokenIndex = position184, tokenIndex184
			return false
		},
		/* 32 keyword <- <((&('N') ('N' 'O' 'T')) | (&('O') ('O' 'R')) | (&('A') ('A' 'N' 'D')))> */
		func() bool {
			position190, tokenIndex190 := position, tokenIndex
			{
				position191 := position
				{
					switch buffer[position] {
					case 'N':
						if buffer[position] != rune('N') {
							goto l190
						}
						position++
						if buffer[position] != rune('O') {
							goto l190
						}
						position++
						if buffer[position] != rune('T') {
							goto l190
						}
						position++
						break
					case 'O':
						if buffer[position] != rune('O') {
							goto l190
						}
						position++
						if buffer[position] != rune('R') {
							goto l190
						}
						position++
						break
					default:
						if buffer[position] != rune('A') {
							goto l190
						}
						position++
						if buffer[position] != rune('N') {
							goto l190
						}
						position++
						if buffer[position] != rune('D') {
							goto l190
						}
						position++
						break
					}
				}

				add(rulekeyword, position191)
			}
			return true
		l190:
			position, tokenIndex = position190, tokenIndex190
			return false
		},
		/* 33 valid_letter <- <(start_letter+ ((&('\\') ('\\' special_char)) | (&('-') '-') | (&('@') '@') | (&('.') '.') | (&('_') '_') | (&('?') '?') | (&('*') '*') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))*)> */
		func() bool {
			position193, tokenIndex193 := position, tokenIndex
			{
				position194 := position
				{
					position197 := position
					{
						switch buffer[position] {
						case '\\':
							if buffer[position] != rune('\\') {
								goto l193
							}
							position++
							if !_rules[rulespecial_char]() {
								goto l193
							}
							break
						case '*':
							if buffer[position] != rune('*') {
								goto l193
							}
							position++
							break
						case '_':
							if buffer[position] != rune('_') {
								goto l193
							}
							position++
							break
						case '.':
							if buffer[position] != rune('.') {
								goto l193
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l193
							}
							position++
							break
						case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l193
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l193
							}
							position++
							break
						}
					}

					add(rulestart_letter, position197)
				}
			l195:
				{
					position196, tokenIndex196 := position, tokenIndex
					{
						position199 := position
						{
							switch buffer[position] {
							case '\\':
								if buffer[position] != rune('\\') {
									goto l196
								}
								position++
								if !_rules[rulespecial_char]() {
									goto l196
								}
								break
							case '*':
								if buffer[position] != rune('*') {
									goto l196
								}
								position++
								break
							case '_':
								if buffer[position] != rune('_') {
									goto l196
								}
								position++
								break
							case '.':
								if buffer[position] != rune('.') {
									goto l196
								}
								position++
								break
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l196
								}
								position++
								break
							case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l196
								}
								position++
								break
							default:
								if c := buffer[position]; c < rune('A') || c > rune('Z') {
									goto l196
								}
								position++
								break
							}
						}

						add(rulestart_letter, position199)
					}
					goto l195
				l196:
					position, tokenIndex = position196, tokenIndex196
				}
			l201:
				{
					position202, tokenIndex202 := position, tokenIndex
					{
						switch buffer[position] {
						case '\\':
							if buffer[position] != rune('\\') {
								goto l202
							}
							position++
							if !_rules[rulespecial_char]() {
								goto l202
							}
							break
						case '-':
							if buffer[position] != rune('-') {
								goto l202
							}
							position++
							break
						case '@':
							if buffer[position] != rune('@') {
								goto l202
							}
							position++
							break
						case '.':
							if buffer[position] != rune('.') {
								goto l202
							}
							position++
							break
						case '_':
							if buffer[position] != rune('_') {
								goto l202
							}
							position++
							break
						case '?':
							if buffer[position] != rune('?') {
								goto l202
							}
							position++
							break
						case '*':
							if buffer[position] != rune('*') {
								goto l202
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l202
							}
							position++
							break
						case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l202
							}
							position++
							break
						default:
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l202
							}
							position++
							break
						}
					}

					goto l201
				l202:
					position, tokenIndex = position202, tokenIndex202
				}
				add(rulevalid_letter, position194)
			}
			return true
		l193:
			position, tokenIndex = position193, tokenIndex193
			return false
		},
		/* 34 start_letter <- <((&('\\') ('\\' special_char)) | (&('*') '*') | (&('_') '_') | (&('.') '.') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))> */
		nil,
		/* 35 end_letter <- <([A-Z] / [a-z] / [0-9] / '*' / '?' / '_' / '.' / ('\\' special_char))> */
		nil,
		/* 36 special_char <- <((&(':') ':') | (&('\\') '\\') | (&('?') '?') | (&('*') '*') | (&('~') '~') | (&('"') '"') | (&('^') '^') | (&(']') ']') | (&('[') '[') | (&('}') '}') | (&('{') '{') | (&(')') ')') | (&('(') '(') | (&('!') '!') | (&('|') '|') | (&('&') '&') | (&('+') '+') | (&('-') '-'))> */
		func() bool {
			position206, tokenIndex206 := position, tokenIndex
			{
				position207 := position
				{
					switch buffer[position] {
					case ':':
						if buffer[position] != rune(':') {
							goto l206
						}
						position++
						break
					case '\\':
						if buffer[position] != rune('\\') {
							goto l206
						}
						position++
						break
					case '?':
						if buffer[position] != rune('?') {
							goto l206
						}
						position++
						break
					case '*':
						if buffer[position] != rune('*') {
							goto l206
						}
						position++
						break
					case '~':
						if buffer[position] != rune('~') {
							goto l206
						}
						position++
						break
					case '"':
						if buffer[position] != rune('"') {
							goto l206
						}
						position++
						break
					case '^':
						if buffer[position] != rune('^') {
							goto l206
						}
						position++
						break
					case ']':
						if buffer[position] != rune(']') {
							goto l206
						}
						position++
						break
					case '[':
						if buffer[position] != rune('[') {
							goto l206
						}
						position++
						break
					case '}':
						if buffer[position] != rune('}') {
							goto l206
						}
						position++
						break
					case '{':
						if buffer[position] != rune('{') {
							goto l206
						}
						position++
						break
					case ')':
						if buffer[position] != rune(')') {
							goto l206
						}
						position++
						break
					case '(':
						if buffer[position] != rune('(') {
							goto l206
						}
						position++
						break
					case '!':
						if buffer[position] != rune('!') {
							goto l206
						}
						position++
						break
					case '|':
						if buffer[position] != rune('|') {
							goto l206
						}
						position++
						break
					case '&':
						if buffer[position] != rune('&') {
							goto l206
						}
						position++
						break
					case '+':
						if buffer[position] != rune('+') {
							goto l206
						}
						position++
						break
					default:
						if buffer[position] != rune('-') {
							goto l206
						}
						position++
						break
					}
				}

				add(rulespecial_char, position207)
			}
			return true
		l206:
			position, tokenIndex = position206, tokenIndex206
			return false
		},
		/* 37 open_paren <- <'('> */
		nil,
		/* 38 close_paren <- <')'> */
		nil,
		/* 39 open_incl <- <'['> */
		nil,
		/* 40 close_incl <- <']'> */
		nil,
		/* 41 open_excl <- <'{'> */
		nil,
		/* 42 close_excl <- <'}'> */
		nil,
		/* 43 space <- <((&('\r') '\r') | (&('\n') '\n') | (&('\t') '\t') | (&(' ') ' '))+> */
		func() bool {
			position215, tokenIndex215 := position, tokenIndex
			{
				position216 := position
				{
					switch buffer[position] {
					case '\r':
						if buffer[position] != rune('\r') {
							goto l215
						}
						position++
						break
					case '\n':
						if buffer[position] != rune('\n') {
							goto l215
						}
						position++
						break
					case '\t':
						if buffer[position] != rune('\t') {
							goto l215
						}
						position++
						break
					default:
						if buffer[position] != rune(' ') {
							goto l215
						}
						position++
						break
					}
				}

			l217:
				{
					position218, tokenIndex218 := position, tokenIndex
					{
						switch buffer[position] {
						case '\r':
							if buffer[position] != rune('\r') {
								goto l218
							}
							position++
							break
						case '\n':
							if buffer[position] != rune('\n') {
								goto l218
							}
							position++
							break
						case '\t':
							if buffer[position] != rune('\t') {
								goto l218
							}
							position++
							break
						default:
							if buffer[position] != rune(' ') {
								goto l218
							}
							position++
							break
						}
					}

					goto l217
				l218:
					position, tokenIndex = position218, tokenIndex218
				}
				add(rulespace, position216)
			}
			return true
		l215:
			position, tokenIndex = position215, tokenIndex215
			return false
		},
		nil,
		/* 46 Action0 <- <{ p.AddTerm(buffer[begin:end]) }> */
		nil,
		/* 47 Action1 <- <{ p.StartBasic() }> */
		nil,
		/* 48 Action2 <- <{ p.StartGrouped() }> */
		nil,
		/* 49 Action3 <- <{ p.SetCompleted() }> */
		nil,
		/* 50 Action4 <- <{ p.StartRange(true) }> */
		nil,
		/* 51 Action5 <- <{ p.StartRange(false) }> */
		nil,
		/* 52 Action6 <- <{ p.AddField(buffer[begin:end]) }> */
		nil,
		/* 53 Action7 <- <{ p.AddRange(buffer[begin:end]) }> */
		nil,
		/* 54 Action8 <- <{ p.StartSubQuery() }> */
		nil,
		/* 55 Action9 <- <{ p.EndSubQuery() }> */
		nil,
		/* 56 Action10 <- <{ p.AddOp(OpBinOr) }> */
		nil,
		/* 57 Action11 <- <{ p.AddOp(OpBinAnd) }> */
		nil,
		/* 58 Action12 <- <{ p.StartSubQuery() }> */
		nil,
		/* 59 Action13 <- <{ p.EndSubQuery() }> */
		nil,
		/* 60 Action14 <- <{ p.AddTermOp(OpUnaryNot) }> */
		nil,
		/* 61 Action15 <- <{ p.AddTermOp(OpUnaryNot) }> */
		nil,
		/* 62 Action16 <- <{ p.AddTermOp(OpUnaryReq) }> */
		nil,
		/* 63 Action17 <- <{ p.AddTermOp(OpUnaryPro) }> */
		nil,
		/* 64 Action18 <- <{ p.AddOp(OpBoost) }> */
		nil,
		/* 65 Action19 <- <{ p.AddOp(OpFuzzy) }> */
		nil,
		/* 66 Action20 <- <{ p.AddTerm(buffer[begin:end]) }> */
		nil,
		/* 67 Action21 <- <{ p.AddTerm(buffer[begin:end]) }> */
		nil,
	}
	p.rules = _rules
}
