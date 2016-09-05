package search

import (
	/*"bytes"*/
	"fmt"
	"math"
	"sort"
	"strconv"
	"unicode"
)

const END_SYMBOL rune = 4

/* The rule types inferred from the grammar are below. */
type Rule uint8

const (
	RuleUnknown Rule = iota
	Ruleq
	Rulebody
	Ruleexpression
	Ruleterm
	Rulefield
	Rulefield_norm
	Rulefield_group
	Rulefield_range
	Rulefield_inc_range
	Rulefield_exc_range
	Rulefield_name
	Rulerange_value
	Rulegroup
	Ruleoperation
	Ruleunary_op
	Rulebinary_op
	Ruleboolean_operator
	Ruleor_operator
	Ruleand_operator
	Rulenot_op
	Rulenot_operator
	Rulebang_operator
	Rulerequired_op
	Rulerequired_operator
	Ruleprohibited_op
	Ruleprohibited_operator
	Ruleboost_op
	Rulefuzzy_op
	Rulefuzzy_param
	Rulestring
	Rulekeyword
	Rulevalid_letter
	Rulestart_letter
	Ruleend_letter
	Rulespecial_char
	Ruleopen_paren
	Ruleclose_paren
	Ruleopen_incl
	Ruleclose_incl
	Ruleopen_excl
	Ruleclose_excl
	Rulespace
	RulePegText
	RuleAction0
	RuleAction1
	RuleAction2
	RuleAction3
	RuleAction4
	RuleAction5
	RuleAction6
	RuleAction7
	RuleAction8
	RuleAction9
	RuleAction10
	RuleAction11
	RuleAction12
	RuleAction13
	RuleAction14
	RuleAction15
	RuleAction16
	RuleAction17
	RuleAction18
	RuleAction19
	RuleAction20

	RulePre_
	Rule_In_
	Rule_Suf
)

var Rul3s = [...]string{
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
	"group",
	"operation",
	"unary_op",
	"binary_op",
	"boolean_operator",
	"or_operator",
	"and_operator",
	"not_op",
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

	"Pre_",
	"_In_",
	"_Suf",
}

type TokenTree interface {
	Print()
	PrintSyntax()
	PrintSyntaxTree(buffer string)
	Add(rule Rule, begin, end, next, depth int)
	Expand(index int) TokenTree
	Tokens() <-chan token32
	Error() []token32
	trim(length int)
}

/* ${@} bit structure for abstract syntax tree */
type token16 struct {
	Rule
	begin, end, next int16
}

func (t *token16) isZero() bool {
	return t.Rule == RuleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token16) isParentOf(u token16) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token16) GetToken32() token32 {
	return token32{Rule: t.Rule, begin: int32(t.begin), end: int32(t.end), next: int32(t.next)}
}

func (t *token16) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", Rul3s[t.Rule], t.begin, t.end, t.next)
}

type tokens16 struct {
	tree    []token16
	ordered [][]token16
}

func (t *tokens16) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens16) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens16) Order() [][]token16 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int16, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.Rule == RuleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token16, len(depths)), make([]token16, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = int16(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type State16 struct {
	token16
	depths []int16
	leaf   bool
}

func (t *tokens16) PreOrder() (<-chan State16, [][]token16) {
	s, ordered := make(chan State16, 6), t.Order()
	go func() {
		var states [8]State16
		for i, _ := range states {
			states[i].depths = make([]int16, len(ordered))
		}
		depths, state, depth := make([]int16, len(ordered)), 0, 1
		write := func(t token16, leaf bool) {
			S := states[state]
			state, S.Rule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.Rule, t.begin, t.end, int16(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token16 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token16{Rule: Rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token16{Rule: RulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.Rule != RuleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.Rule != RuleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token16{Rule: Rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens16) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", Rul3s[ordered[i][depths[i]-1].Rule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", Rul3s[token.Rule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", Rul3s[ordered[i][depths[i]-1].Rule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", Rul3s[token.Rule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", Rul3s[ordered[i][depths[i]-1].Rule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", Rul3s[token.Rule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens16) PrintSyntaxTree(buffer2 string) {
	buffer := []rune(buffer2)
	fmt.Printf("And the buffer here? '%s'\n", buffer)
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", Rul3s[token.Rule], strconv.Quote(string(buffer[token.begin:token.end])))
	}
}

func (t *tokens16) Add(rule Rule, begin, end, depth, index int) {
	t.tree[index] = token16{Rule: rule, begin: int16(begin), end: int16(end), next: int16(depth)}
}

func (t *tokens16) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.GetToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens16) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].GetToken32()
		}
	}
	return tokens
}

/* ${@} bit structure for abstract syntax tree */
type token32 struct {
	Rule
	begin, end, next int32
}

func (t *token32) isZero() bool {
	return t.Rule == RuleUnknown && t.begin == 0 && t.end == 0 && t.next == 0
}

func (t *token32) isParentOf(u token32) bool {
	return t.begin <= u.begin && t.end >= u.end && t.next > u.next
}

func (t *token32) GetToken32() token32 {
	return token32{Rule: t.Rule, begin: int32(t.begin), end: int32(t.end), next: int32(t.next)}
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v %v", Rul3s[t.Rule], t.begin, t.end, t.next)
}

type tokens32 struct {
	tree    []token32
	ordered [][]token32
}

func (t *tokens32) trim(length int) {
	t.tree = t.tree[0:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) Order() [][]token32 {
	if t.ordered != nil {
		return t.ordered
	}

	depths := make([]int32, 1, math.MaxInt16)
	for i, token := range t.tree {
		if token.Rule == RuleUnknown {
			t.tree = t.tree[:i]
			break
		}
		depth := int(token.next)
		if length := len(depths); depth >= length {
			depths = depths[:depth+1]
		}
		depths[depth]++
	}
	depths = append(depths, 0)

	ordered, pool := make([][]token32, len(depths)), make([]token32, len(t.tree)+len(depths))
	for i, depth := range depths {
		depth++
		ordered[i], pool, depths[i] = pool[:depth], pool[depth:], 0
	}

	for i, token := range t.tree {
		depth := token.next
		token.next = int32(i)
		ordered[depth][depths[depth]] = token
		depths[depth]++
	}
	t.ordered = ordered
	return ordered
}

type State32 struct {
	token32
	depths []int32
	leaf   bool
}

func (t *tokens32) PreOrder() (<-chan State32, [][]token32) {
	s, ordered := make(chan State32, 6), t.Order()
	go func() {
		var states [8]State32
		for i, _ := range states {
			states[i].depths = make([]int32, len(ordered))
		}
		depths, state, depth := make([]int32, len(ordered)), 0, 1
		write := func(t token32, leaf bool) {
			S := states[state]
			state, S.Rule, S.begin, S.end, S.next, S.leaf = (state+1)%8, t.Rule, t.begin, t.end, int32(depth), leaf
			copy(S.depths, depths)
			s <- S
		}

		states[state].token32 = ordered[0][0]
		depths[0]++
		state++
		a, b := ordered[depth-1][depths[depth-1]-1], ordered[depth][depths[depth]]
	depthFirstSearch:
		for {
			for {
				if i := depths[depth]; i > 0 {
					if c, j := ordered[depth][i-1], depths[depth-1]; a.isParentOf(c) &&
						(j < 2 || !ordered[depth-1][j-2].isParentOf(c)) {
						if c.end != b.begin {
							write(token32{Rule: Rule_In_, begin: c.end, end: b.begin}, true)
						}
						break
					}
				}

				if a.begin < b.begin {
					write(token32{Rule: RulePre_, begin: a.begin, end: b.begin}, true)
				}
				break
			}

			next := depth + 1
			if c := ordered[next][depths[next]]; c.Rule != RuleUnknown && b.isParentOf(c) {
				write(b, false)
				depths[depth]++
				depth, a, b = next, b, c
				continue
			}

			write(b, true)
			depths[depth]++
			c, parent := ordered[depth][depths[depth]], true
			for {
				if c.Rule != RuleUnknown && a.isParentOf(c) {
					b = c
					continue depthFirstSearch
				} else if parent && b.end != a.end {
					write(token32{Rule: Rule_Suf, begin: b.end, end: a.end}, true)
				}

				depth--
				if depth > 0 {
					a, b, c = ordered[depth-1][depths[depth-1]-1], a, ordered[depth][depths[depth]]
					parent = a.isParentOf(b)
					continue
				}

				break depthFirstSearch
			}
		}

		close(s)
	}()
	return s, ordered
}

func (t *tokens32) PrintSyntax() {
	tokens, ordered := t.PreOrder()
	max := -1
	for token := range tokens {
		if !token.leaf {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[36m%v\x1B[m", Rul3s[ordered[i][depths[i]-1].Rule])
			}
			fmt.Printf(" \x1B[36m%v\x1B[m\n", Rul3s[token.Rule])
		} else if token.begin == token.end {
			fmt.Printf("%v", token.begin)
			for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
				fmt.Printf(" \x1B[31m%v\x1B[m", Rul3s[ordered[i][depths[i]-1].Rule])
			}
			fmt.Printf(" \x1B[31m%v\x1B[m\n", Rul3s[token.Rule])
		} else {
			for c, end := token.begin, token.end; c < end; c++ {
				if i := int(c); max+1 < i {
					for j := max; j < i; j++ {
						fmt.Printf("skip %v %v\n", j, token.String())
					}
					max = i
				} else if i := int(c); i <= max {
					for j := i; j <= max; j++ {
						fmt.Printf("dupe %v %v\n", j, token.String())
					}
				} else {
					max = int(c)
				}
				fmt.Printf("%v", c)
				for i, leaf, depths := 0, int(token.next), token.depths; i < leaf; i++ {
					fmt.Printf(" \x1B[34m%v\x1B[m", Rul3s[ordered[i][depths[i]-1].Rule])
				}
				fmt.Printf(" \x1B[34m%v\x1B[m\n", Rul3s[token.Rule])
			}
			fmt.Printf("\n")
		}
	}
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	fmt.Printf("And the buffer here? '%s'\n", buffer)
	tokens, _ := t.PreOrder()
	for token := range tokens {
		for c := 0; c < int(token.next); c++ {
			fmt.Printf(" ")
		}
		fmt.Printf("\x1B[34m%v\x1B[m %v\n", Rul3s[token.Rule], strconv.Quote(buffer[token.begin:token.end]))
	}
}

func (t *tokens32) Add(rule Rule, begin, end, depth, index int) {
	t.tree[index] = token32{Rule: rule, begin: int32(begin), end: int32(end), next: int32(depth)}
}

func (t *tokens32) Tokens() <-chan token32 {
	s := make(chan token32, 16)
	go func() {
		for _, v := range t.tree {
			s <- v.GetToken32()
		}
		close(s)
	}()
	return s
}

func (t *tokens32) Error() []token32 {
	ordered := t.Order()
	length := len(ordered)
	tokens, length := make([]token32, length), length-1
	for i, _ := range tokens {
		o := ordered[length-i]
		if len(o) > 1 {
			tokens[i] = o[len(o)-2].GetToken32()
		}
	}
	return tokens
}

func (t *tokens16) Expand(index int) TokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		for i, v := range tree {
			expanded[i] = v.GetToken32()
		}
		return &tokens32{tree: expanded}
	}
	return nil
}

func (t *tokens32) Expand(index int) TokenTree {
	tree := t.tree
	if index >= len(tree) {
		expanded := make([]token32, 2*len(tree))
		copy(expanded, tree)
		t.tree = expanded
	}
	return nil
}

type Tokenizer struct {
	Token

	Buffer string
	buffer []rune
	rules  [65]func() bool
	Parse  func(rule ...int) error
	Reset  func()
	TokenTree
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer string, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer[0:] {
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
	p *Tokenizer
}

func (e *parseError) Error() string {
	tokens, error := e.p.TokenTree.Error(), "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.Buffer, positions)
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		error += fmt.Sprintf("parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n",
			Rul3s[token.Rule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			/*strconv.Quote(*/ string(e.p.buffer[begin:end]) /*)*/)
	}

	return error
}

func (p *Tokenizer) PrintSyntaxTree() {
	p.TokenTree.PrintSyntaxTree(p.Buffer)
}

func (p *Tokenizer) Highlighter() {
	p.TokenTree.PrintSyntax()
}

func isASCII(str string) bool {
	for _, r := range []rune(str) {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

func (p *Tokenizer) Execute() {
	buffer, begin, end := p.buffer, 0, 0
	p.PrintSyntaxTree()
	
	for token := range p.TokenTree.Tokens() {
		switch token.Rule {
		case RulePegText:
			begin, end = int(token.begin), int(token.end)
		case RuleAction0:
			p.AddTerm(string(buffer[begin:end]))
		case RuleAction1:
			p.StartBasic()
		case RuleAction2:
			p.StartGrouped()
		case RuleAction3:
			p.SetCompleted()
		case RuleAction4:
			p.StartRange(true)
		case RuleAction5:
			p.StartRange(false)
		case RuleAction6:
			p.AddField(string(buffer[begin:end]))
		case RuleAction7:
			p.AddRange(string(buffer[begin:end]))
		case RuleAction8:
			p.StartSubQuery()
		case RuleAction9:
			p.EndSubQuery()
		case RuleAction10:
			p.StartBasic()
		case RuleAction11:
			p.AddOp(OpBinOr)
		case RuleAction12:
			p.AddOp(OpBinAnd)
		case RuleAction13:
			p.AddTermOp(OpUnaryNot)
		case RuleAction14:
			p.AddTermOp(OpUnaryNot)
		case RuleAction15:
			p.AddTermOp(OpUnaryReq)
		case RuleAction16:
			p.AddTermOp(OpUnaryPro)
		case RuleAction17:
			p.AddOp(OpBoost)
		case RuleAction18:
			p.AddOp(OpFuzzy)
		case RuleAction19:
			p.AddTerm(string(buffer[begin:end]))
		case RuleAction20:
			p.AddTerm(string(buffer[begin:end]))

		}
	}
}

func (p *Tokenizer) Init() {
	fmt.Printf("p.Buffer: %s\n", p.Buffer)
	fmt.Printf("p.Buffer as rune array: %v\n", []rune(p.Buffer))
	p.buffer = []rune(p.Buffer)
	if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != END_SYMBOL {
		p.buffer = append(p.buffer, END_SYMBOL)
	}
	fmt.Printf("post-maul p.buffer: %v\n", p.buffer)

	var tree TokenTree = &tokens16{tree: make([]token16, math.MaxInt16)}
	position, depth, tokenIndex, buffer, rules := 0, 0, 0, p.buffer, p.rules

	p.Parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.TokenTree = tree
		if matches {
			p.TokenTree.trim(tokenIndex)
			return nil
		}
		return &parseError{p}
	}

	p.Reset = func() {
		position, tokenIndex, depth = 0, 0, 0
	}

	add := func(rule Rule, begin int) {
		if t := tree.Expand(tokenIndex); t != nil {
			tree = t
		}
		tree.Add(rule, begin, position, depth, tokenIndex)
		tokenIndex++
	}

	matchDot := func() bool {
		if buffer[position] != END_SYMBOL {
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

	rules = [...]func() bool{
		nil,
		/* 0 q <- <(body !.)> */
		func() bool {
			position0, tokenIndex0, depth0 := position, tokenIndex, depth
			{
				position1 := position
				depth++
				if !rules[Rulebody]() {
					goto l0
				}
				{
					position2, tokenIndex2, depth2 := position, tokenIndex, depth
					if !matchDot() {
						goto l2
					}
					goto l0
				l2:
					position, tokenIndex, depth = position2, tokenIndex2, depth2
				}
				depth--
				add(Ruleq, position1)
			}
			return true
		l0:
			position, tokenIndex, depth = position0, tokenIndex0, depth0
			return false
		},
		/* 1 body <- <(expression / space)*> */
		func() bool {
			{
				position4 := position
				depth++
			l5:
				{
					position6, tokenIndex6, depth6 := position, tokenIndex, depth
					{
						position7, tokenIndex7, depth7 := position, tokenIndex, depth
						{
							position9 := position
							depth++
							{
								position10, tokenIndex10, depth10 := position, tokenIndex, depth
								{
									position12 := position
									depth++
									{
										position13, tokenIndex13, depth13 := position, tokenIndex, depth
										{
											position15 := position
											depth++
											{
												position16, tokenIndex16, depth16 := position, tokenIndex, depth
												if !rules[Rulegroup]() {
													goto l17
												}
												goto l16
											l17:
												position, tokenIndex, depth = position16, tokenIndex16, depth16
												if !rules[Rulefield]() {
													goto l18
												}
												goto l16
											l18:
												position, tokenIndex, depth = position16, tokenIndex16, depth16
												if !rules[Rulefield_range]() {
													goto l19
												}
												goto l16
											l19:
												position, tokenIndex, depth = position16, tokenIndex16, depth16
												if !rules[Ruleterm]() {
													goto l14
												}
											}
										l16:
											{
												position20, tokenIndex20, depth20 := position, tokenIndex, depth
												if !rules[Rulespace]() {
													goto l20
												}
												goto l21
											l20:
												position, tokenIndex, depth = position20, tokenIndex20, depth20
											}
										l21:
											{
												position22 := position
												depth++
												{
													position23, tokenIndex23, depth23 := position, tokenIndex, depth
													{
														position25 := position
														depth++
														{
															position26, tokenIndex26, depth26 := position, tokenIndex, depth
															if buffer[position] != rune('O') {
																goto l27
															}
															position++
															if buffer[position] != rune('R') {
																goto l27
															}
															position++
															goto l26
														l27:
															position, tokenIndex, depth = position26, tokenIndex26, depth26
															if buffer[position] != rune('|') {
																goto l24
															}
															position++
															if buffer[position] != rune('|') {
																goto l24
															}
															position++
														}
													l26:
														depth--
														add(Ruleor_operator, position25)
													}
													{
														add(RuleAction11, position)
													}
													goto l23
												l24:
													position, tokenIndex, depth = position23, tokenIndex23, depth23
													{
														position29 := position
														depth++
														{
															position30, tokenIndex30, depth30 := position, tokenIndex, depth
															if buffer[position] != rune('A') {
																goto l31
															}
															position++
															if buffer[position] != rune('N') {
																goto l31
															}
															position++
															if buffer[position] != rune('D') {
																goto l31
															}
															position++
															goto l30
														l31:
															position, tokenIndex, depth = position30, tokenIndex30, depth30
															if buffer[position] != rune('&') {
																goto l14
															}
															position++
															if buffer[position] != rune('&') {
																goto l14
															}
															position++
														}
													l30:
														depth--
														add(Ruleand_operator, position29)
													}
													{
														add(RuleAction12, position)
													}
												}
											l23:
												depth--
												add(Ruleboolean_operator, position22)
											}
											if !rules[Rulespace]() {
												goto l14
											}
										l33:
											{
												position34, tokenIndex34, depth34 := position, tokenIndex, depth
												if !rules[Rulespace]() {
													goto l34
												}
												goto l33
											l34:
												position, tokenIndex, depth = position34, tokenIndex34, depth34
											}
											if !rules[Rulebody]() {
												goto l14
											}
											depth--
											add(Rulebinary_op, position15)
										}
										goto l13
									l14:
										position, tokenIndex, depth = position13, tokenIndex13, depth13
										{
											position36 := position
											depth++
											{
												switch buffer[position] {
												case '-':
													{
														position38 := position
														depth++
														{
															position39, tokenIndex39, depth39 := position, tokenIndex, depth
															if !rules[Rulevalid_letter]() {
																goto l39
															}
															goto l35
														l39:
															position, tokenIndex, depth = position39, tokenIndex39, depth39
														}
														{
															position40 := position
															depth++
															if buffer[position] != rune('-') {
																goto l35
															}
															position++
															{
																add(RuleAction16, position)
															}
															depth--
															add(Ruleprohibited_operator, position40)
														}
														{
															position42, tokenIndex42, depth42 := position, tokenIndex, depth
															if !rules[Rulefield]() {
																goto l43
															}
															goto l42
														l43:
															position, tokenIndex, depth = position42, tokenIndex42, depth42
															if !rules[Rulefield_range]() {
																goto l44
															}
															goto l42
														l44:
															position, tokenIndex, depth = position42, tokenIndex42, depth42
															if !rules[Ruleterm]() {
																goto l45
															}
															goto l42
														l45:
															position, tokenIndex, depth = position42, tokenIndex42, depth42
															if !rules[Rulestring]() {
																goto l35
															}
														}
													l42:
														depth--
														add(Ruleprohibited_op, position38)
													}
													break
												case '+':
													{
														position46 := position
														depth++
														{
															position47, tokenIndex47, depth47 := position, tokenIndex, depth
															{
																position49, tokenIndex49, depth49 := position, tokenIndex, depth
																if !rules[Rulevalid_letter]() {
																	goto l49
																}
																goto l48
															l49:
																position, tokenIndex, depth = position49, tokenIndex49, depth49
															}
															if !rules[Rulerequired_operator]() {
																goto l48
															}
															{
																position50, tokenIndex50, depth50 := position, tokenIndex, depth
																if !rules[Ruleterm]() {
																	goto l51
																}
																goto l50
															l51:
																position, tokenIndex, depth = position50, tokenIndex50, depth50
																if !rules[Rulestring]() {
																	goto l48
																}
															}
														l50:
															goto l47
														l48:
															position, tokenIndex, depth = position47, tokenIndex47, depth47
															if !rules[Rulerequired_operator]() {
																goto l35
															}
															{
																position52, tokenIndex52, depth52 := position, tokenIndex, depth
																if !rules[Ruleterm]() {
																	goto l53
																}
																goto l52
															l53:
																position, tokenIndex, depth = position52, tokenIndex52, depth52
																if !rules[Rulestring]() {
																	goto l35
																}
															}
														l52:
														}
													l47:
														depth--
														add(Rulerequired_op, position46)
													}
													break
												default:
													{
														add(RuleAction10, position)
													}
													{
														position55 := position
														depth++
														{
															position56, tokenIndex56, depth56 := position, tokenIndex, depth
															{
																position58 := position
																depth++
																if buffer[position] != rune('N') {
																	goto l57
																}
																position++
																if buffer[position] != rune('O') {
																	goto l57
																}
																position++
																if buffer[position] != rune('T') {
																	goto l57
																}
																position++
																{
																	add(RuleAction13, position)
																}
																depth--
																add(Rulenot_operator, position58)
															}
															if !rules[Rulespace]() {
																goto l57
															}
															{
																position60, tokenIndex60, depth60 := position, tokenIndex, depth
																if !rules[Rulefield]() {
																	goto l61
																}
																goto l60
															l61:
																position, tokenIndex, depth = position60, tokenIndex60, depth60
																if !rules[Rulefield_range]() {
																	goto l62
																}
																goto l60
															l62:
																position, tokenIndex, depth = position60, tokenIndex60, depth60
																{
																	switch buffer[position] {
																	case '"':
																		if !rules[Rulestring]() {
																			goto l57
																		}
																		break
																	case '\t', '\n', '\r', ' ', '(':
																		if !rules[Rulegroup]() {
																			goto l57
																		}
																		break
																	default:
																		if !rules[Ruleterm]() {
																			goto l57
																		}
																		break
																	}
																}

															}
														l60:
															goto l56
														l57:
															position, tokenIndex, depth = position56, tokenIndex56, depth56
															{
																position64 := position
																depth++
																if buffer[position] != rune('!') {
																	goto l35
																}
																position++
																{
																	add(RuleAction14, position)
																}
																depth--
																add(Rulebang_operator, position64)
															}
															{
																position66, tokenIndex66, depth66 := position, tokenIndex, depth
																if !rules[Rulespace]() {
																	goto l66
																}
																goto l67
															l66:
																position, tokenIndex, depth = position66, tokenIndex66, depth66
															}
														l67:
															{
																position68, tokenIndex68, depth68 := position, tokenIndex, depth
																if !rules[Rulefield]() {
																	goto l69
																}
																goto l68
															l69:
																position, tokenIndex, depth = position68, tokenIndex68, depth68
																if !rules[Rulefield_range]() {
																	goto l70
																}
																goto l68
															l70:
																position, tokenIndex, depth = position68, tokenIndex68, depth68
																{
																	switch buffer[position] {
																	case '"':
																		if !rules[Rulestring]() {
																			goto l35
																		}
																		break
																	case '\t', '\n', '\r', ' ', '(':
																		if !rules[Rulegroup]() {
																			goto l35
																		}
																		break
																	default:
																		if !rules[Ruleterm]() {
																			goto l35
																		}
																		break
																	}
																}

															}
														l68:
														}
													l56:
														depth--
														add(Rulenot_op, position55)
													}
													break
												}
											}

											depth--
											add(Ruleunary_op, position36)
										}
										goto l13
									l35:
										position, tokenIndex, depth = position13, tokenIndex13, depth13
										{
											position73 := position
											depth++
											{
												position74, tokenIndex74, depth74 := position, tokenIndex, depth
												if !rules[Ruleterm]() {
													goto l75
												}
												goto l74
											l75:
												position, tokenIndex, depth = position74, tokenIndex74, depth74
												if !rules[Rulestring]() {
													goto l72
												}
											}
										l74:
											if buffer[position] != rune('~') {
												goto l72
											}
											position++
											{
												add(RuleAction18, position)
											}
											{
												position77, tokenIndex77, depth77 := position, tokenIndex, depth
												if !rules[Rulefuzzy_param]() {
													goto l77
												}
												goto l78
											l77:
												position, tokenIndex, depth = position77, tokenIndex77, depth77
											}
										l78:
											{
												position79, tokenIndex79, depth79 := position, tokenIndex, depth
												if !rules[Rulespace]() {
													goto l80
												}
												goto l79
											l80:
												position, tokenIndex, depth = position79, tokenIndex79, depth79
												{
													position81, tokenIndex81, depth81 := position, tokenIndex, depth
													if !rules[Rulevalid_letter]() {
														goto l81
													}
													goto l72
												l81:
													position, tokenIndex, depth = position81, tokenIndex81, depth81
												}
											}
										l79:
											depth--
											add(Rulefuzzy_op, position73)
										}
										goto l13
									l72:
										position, tokenIndex, depth = position13, tokenIndex13, depth13
										{
											position82 := position
											depth++
											{
												position83, tokenIndex83, depth83 := position, tokenIndex, depth
												if !rules[Ruleterm]() {
													goto l84
												}
												goto l83
											l84:
												position, tokenIndex, depth = position83, tokenIndex83, depth83
												if !rules[Rulestring]() {
													goto l11
												}
											}
										l83:
											if buffer[position] != rune('^') {
												goto l11
											}
											position++
											{
												add(RuleAction17, position)
											}
											if !rules[Rulefuzzy_param]() {
												goto l11
											}
											depth--
											add(Ruleboost_op, position82)
										}
									}
								l13:
									depth--
									add(Ruleoperation, position12)
								}
								goto l10
							l11:
								position, tokenIndex, depth = position10, tokenIndex10, depth10
								if !rules[Rulefield]() {
									goto l86
								}
								goto l10
							l86:
								position, tokenIndex, depth = position10, tokenIndex10, depth10
								if !rules[Rulefield_range]() {
									goto l87
								}
								goto l10
							l87:
								position, tokenIndex, depth = position10, tokenIndex10, depth10
								{
									switch buffer[position] {
									case '"':
										if !rules[Rulestring]() {
											goto l8
										}
										break
									case '\t', '\n', '\r', ' ', '(':
										if !rules[Rulegroup]() {
											goto l8
										}
										break
									default:
										if !rules[Ruleterm]() {
											goto l8
										}
										break
									}
								}

							}
						l10:
							depth--
							add(Ruleexpression, position9)
						}
						goto l7
					l8:
						position, tokenIndex, depth = position7, tokenIndex7, depth7
						if !rules[Rulespace]() {
							goto l6
						}
					}
				l7:
					goto l5
				l6:
					position, tokenIndex, depth = position6, tokenIndex6, depth6
				}
				depth--
				add(Rulebody, position4)
			}
			return true
		},
		/* 2 expression <- <(operation / field / field_range / ((&('"') string) | (&('\t' | '\n' | '\r' | ' ' | '(') group) | (&('*' | '.' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | '\\' | '_' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') term)))> */
		nil,
		/* 3 term <- <(<((keyword valid_letter+) / (!keyword !'?' valid_letter))> Action0)> */
		func() bool {
			position90, tokenIndex90, depth90 := position, tokenIndex, depth
			{
				position91 := position
				depth++
				{
					position92 := position
					depth++
					{
						position93, tokenIndex93, depth93 := position, tokenIndex, depth
						if !rules[Rulekeyword]() {
							goto l94
						}
						if !rules[Rulevalid_letter]() {
							goto l94
						}
					l95:
						{
							position96, tokenIndex96, depth96 := position, tokenIndex, depth
							if !rules[Rulevalid_letter]() {
								goto l96
							}
							goto l95
						l96:
							position, tokenIndex, depth = position96, tokenIndex96, depth96
						}
						goto l93
					l94:
						position, tokenIndex, depth = position93, tokenIndex93, depth93
						{
							position97, tokenIndex97, depth97 := position, tokenIndex, depth
							if !rules[Rulekeyword]() {
								goto l97
							}
							goto l90
						l97:
							position, tokenIndex, depth = position97, tokenIndex97, depth97
						}
						{
							position98, tokenIndex98, depth98 := position, tokenIndex, depth
							if buffer[position] != rune('?') {
								goto l98
							}
							position++
							goto l90
						l98:
							position, tokenIndex, depth = position98, tokenIndex98, depth98
						}
						if !rules[Rulevalid_letter]() {
							goto l90
						}
					}
				l93:
					depth--
					add(RulePegText, position92)
				}
				{
					add(RuleAction0, position)
				}
				depth--
				add(Ruleterm, position91)
			}
			return true
		l90:
			position, tokenIndex, depth = position90, tokenIndex90, depth90
			return false
		},
		/* 4 field <- <(field_norm / field_group)> */
		func() bool {
			position100, tokenIndex100, depth100 := position, tokenIndex, depth
			{
				position101 := position
				depth++
				{
					position102, tokenIndex102, depth102 := position, tokenIndex, depth
					{
						position104 := position
						depth++
						{
							add(RuleAction1, position)
						}
						if !rules[Rulefield_name]() {
							goto l103
						}
						if buffer[position] != rune(':') {
							goto l103
						}
						position++
						{
							position106, tokenIndex106, depth106 := position, tokenIndex, depth
							if !rules[Ruleterm]() {
								goto l107
							}
							goto l106
						l107:
							position, tokenIndex, depth = position106, tokenIndex106, depth106
							if !rules[Rulestring]() {
								goto l103
							}
						}
					l106:
						depth--
						add(Rulefield_norm, position104)
					}
					goto l102
				l103:
					position, tokenIndex, depth = position102, tokenIndex102, depth102
					{
						position108 := position
						depth++
						{
							add(RuleAction2, position)
						}
						if !rules[Rulefield_name]() {
							goto l100
						}
						if buffer[position] != rune(':') {
							goto l100
						}
						position++
						if !rules[Rulegroup]() {
							goto l100
						}
						{
							add(RuleAction3, position)
						}
						depth--
						add(Rulefield_group, position108)
					}
				}
			l102:
				depth--
				add(Rulefield, position101)
			}
			return true
		l100:
			position, tokenIndex, depth = position100, tokenIndex100, depth100
			return false
		},
		/* 5 field_norm <- <(Action1 field_name ':' (term / string))> */
		nil,
		/* 6 field_group <- <(Action2 field_name ':' group Action3)> */
		nil,
		/* 7 field_range <- <(field_inc_range / field_exc_range)> */
		func() bool {
			position113, tokenIndex113, depth113 := position, tokenIndex, depth
			{
				position114 := position
				depth++
				{
					position115, tokenIndex115, depth115 := position, tokenIndex, depth
					{
						position117 := position
						depth++
						{
							add(RuleAction4, position)
						}
						if !rules[Rulefield_name]() {
							goto l116
						}
						if buffer[position] != rune(':') {
							goto l116
						}
						position++
						{
							position119 := position
							depth++
							if buffer[position] != rune('[') {
								goto l116
							}
							position++
							depth--
							add(Ruleopen_incl, position119)
						}
						if !rules[Rulerange_value]() {
							goto l116
						}
						if buffer[position] != rune(' ') {
							goto l116
						}
						position++
						if buffer[position] != rune('T') {
							goto l116
						}
						position++
						if buffer[position] != rune('O') {
							goto l116
						}
						position++
						if buffer[position] != rune(' ') {
							goto l116
						}
						position++
						if !rules[Rulerange_value]() {
							goto l116
						}
						{
							position120 := position
							depth++
							if buffer[position] != rune(']') {
								goto l116
							}
							position++
							depth--
							add(Ruleclose_incl, position120)
						}
						depth--
						add(Rulefield_inc_range, position117)
					}
					goto l115
				l116:
					position, tokenIndex, depth = position115, tokenIndex115, depth115
					{
						position121 := position
						depth++
						{
							add(RuleAction5, position)
						}
						if !rules[Rulefield_name]() {
							goto l113
						}
						if buffer[position] != rune(':') {
							goto l113
						}
						position++
						{
							position123 := position
							depth++
							if buffer[position] != rune('{') {
								goto l113
							}
							position++
							depth--
							add(Ruleopen_excl, position123)
						}
						if !rules[Rulerange_value]() {
							goto l113
						}
						if buffer[position] != rune(' ') {
							goto l113
						}
						position++
						if buffer[position] != rune('T') {
							goto l113
						}
						position++
						if buffer[position] != rune('O') {
							goto l113
						}
						position++
						if buffer[position] != rune(' ') {
							goto l113
						}
						position++
						if !rules[Rulerange_value]() {
							goto l113
						}
						{
							position124 := position
							depth++
							if buffer[position] != rune('}') {
								goto l113
							}
							position++
							depth--
							add(Ruleclose_excl, position124)
						}
						depth--
						add(Rulefield_exc_range, position121)
					}
				}
			l115:
				depth--
				add(Rulefield_range, position114)
			}
			return true
		l113:
			position, tokenIndex, depth = position113, tokenIndex113, depth113
			return false
		},
		/* 8 field_inc_range <- <(Action4 field_name ':' open_incl range_value (' ' 'T' 'O' ' ') range_value close_incl)> */
		nil,
		/* 9 field_exc_range <- <(Action5 field_name ':' open_excl range_value (' ' 'T' 'O' ' ') range_value close_excl)> */
		nil,
		/* 10 field_name <- <(<(!keyword valid_letter+)> Action6)> */
		func() bool {
			position127, tokenIndex127, depth127 := position, tokenIndex, depth
			{
				position128 := position
				depth++
				{
					position129 := position
					depth++
					{
						position130, tokenIndex130, depth130 := position, tokenIndex, depth
						if !rules[Rulekeyword]() {
							goto l130
						}
						goto l127
					l130:
						position, tokenIndex, depth = position130, tokenIndex130, depth130
					}
					if !rules[Rulevalid_letter]() {
						goto l127
					}
				l131:
					{
						position132, tokenIndex132, depth132 := position, tokenIndex, depth
						if !rules[Rulevalid_letter]() {
							goto l132
						}
						goto l131
					l132:
						position, tokenIndex, depth = position132, tokenIndex132, depth132
					}
					depth--
					add(RulePegText, position129)
				}
				{
					add(RuleAction6, position)
				}
				depth--
				add(Rulefield_name, position128)
			}
			return true
		l127:
			position, tokenIndex, depth = position127, tokenIndex127, depth127
			return false
		},
		/* 11 range_value <- <(<(valid_letter+ / '*')> Action7)> */
		func() bool {
			position134, tokenIndex134, depth134 := position, tokenIndex, depth
			{
				position135 := position
				depth++
				{
					position136 := position
					depth++
					{
						position137, tokenIndex137, depth137 := position, tokenIndex, depth
						if !rules[Rulevalid_letter]() {
							goto l138
						}
					l139:
						{
							position140, tokenIndex140, depth140 := position, tokenIndex, depth
							if !rules[Rulevalid_letter]() {
								goto l140
							}
							goto l139
						l140:
							position, tokenIndex, depth = position140, tokenIndex140, depth140
						}
						goto l137
					l138:
						position, tokenIndex, depth = position137, tokenIndex137, depth137
						if buffer[position] != rune('*') {
							goto l134
						}
						position++
					}
				l137:
					depth--
					add(RulePegText, position136)
				}
				{
					add(RuleAction7, position)
				}
				depth--
				add(Rulerange_value, position135)
			}
			return true
		l134:
			position, tokenIndex, depth = position134, tokenIndex134, depth134
			return false
		},
		/* 12 group <- <(space? Action8 open_paren body close_paren Action9 space?)> */
		func() bool {
			position142, tokenIndex142, depth142 := position, tokenIndex, depth
			{
				position143 := position
				depth++
				{
					position144, tokenIndex144, depth144 := position, tokenIndex, depth
					if !rules[Rulespace]() {
						goto l144
					}
					goto l145
				l144:
					position, tokenIndex, depth = position144, tokenIndex144, depth144
				}
			l145:
				{
					add(RuleAction8, position)
				}
				{
					position147 := position
					depth++
					if buffer[position] != rune('(') {
						goto l142
					}
					position++
					depth--
					add(Ruleopen_paren, position147)
				}
				if !rules[Rulebody]() {
					goto l142
				}
				{
					position148 := position
					depth++
					if buffer[position] != rune(')') {
						goto l142
					}
					position++
					depth--
					add(Ruleclose_paren, position148)
				}
				{
					add(RuleAction9, position)
				}
				{
					position150, tokenIndex150, depth150 := position, tokenIndex, depth
					if !rules[Rulespace]() {
						goto l150
					}
					goto l151
				l150:
					position, tokenIndex, depth = position150, tokenIndex150, depth150
				}
			l151:
				depth--
				add(Rulegroup, position143)
			}
			return true
		l142:
			position, tokenIndex, depth = position142, tokenIndex142, depth142
			return false
		},
		/* 13 operation <- <(binary_op / unary_op / fuzzy_op / boost_op)> */
		nil,
		/* 14 unary_op <- <((&('-') prohibited_op) | (&('+') required_op) | (&('!' | 'N') (Action10 not_op)))> */
		nil,
		/* 15 binary_op <- <((group / field / field_range / term) space? boolean_operator space+ body)> */
		nil,
		/* 16 boolean_operator <- <((or_operator Action11) / (and_operator Action12))> */
		nil,
		/* 17 or_operator <- <(('O' 'R') / ('|' '|'))> */
		nil,
		/* 18 and_operator <- <(('A' 'N' 'D') / ('&' '&'))> */
		nil,
		/* 19 not_op <- <((not_operator space (field / field_range / ((&('"') string) | (&('\t' | '\n' | '\r' | ' ' | '(') group) | (&('*' | '.' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | '\\' | '_' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') term)))) / (bang_operator space? (field / field_range / ((&('"') string) | (&('\t' | '\n' | '\r' | ' ' | '(') group) | (&('*' | '.' | '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9' | 'A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z' | '\\' | '_' | 'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') term)))))> */
		nil,
		/* 20 not_operator <- <('N' 'O' 'T' Action13)> */
		nil,
		/* 21 bang_operator <- <('!' Action14)> */
		nil,
		/* 22 required_op <- <((!valid_letter required_operator (term / string)) / (required_operator (term / string)))> */
		nil,
		/* 23 required_operator <- <('+' Action15)> */
		func() bool {
			position162, tokenIndex162, depth162 := position, tokenIndex, depth
			{
				position163 := position
				depth++
				if buffer[position] != rune('+') {
					goto l162
				}
				position++
				{
					add(RuleAction15, position)
				}
				depth--
				add(Rulerequired_operator, position163)
			}
			return true
		l162:
			position, tokenIndex, depth = position162, tokenIndex162, depth162
			return false
		},
		/* 24 prohibited_op <- <(!valid_letter prohibited_operator (field / field_range / term / string))> */
		nil,
		/* 25 prohibited_operator <- <('-' Action16)> */
		nil,
		/* 26 boost_op <- <((term / string) '^' Action17 fuzzy_param)> */
		nil,
		/* 27 fuzzy_op <- <((term / string) '~' Action18 fuzzy_param? (space / !valid_letter))> */
		nil,
		/* 28 fuzzy_param <- <(<(([0-9] ('.' '?') [0-9]) / [0-9]+)> Action19)> */
		func() bool {
			position169, tokenIndex169, depth169 := position, tokenIndex, depth
			{
				position170 := position
				depth++
				{
					position171 := position
					depth++
					{
						position172, tokenIndex172, depth172 := position, tokenIndex, depth
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l173
						}
						position++
						if buffer[position] != rune('.') {
							goto l173
						}
						position++
						if buffer[position] != rune('?') {
							goto l173
						}
						position++
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l173
						}
						position++
						goto l172
					l173:
						position, tokenIndex, depth = position172, tokenIndex172, depth172
						if c := buffer[position]; c < rune('0') || c > rune('9') {
							goto l169
						}
						position++
					l174:
						{
							position175, tokenIndex175, depth175 := position, tokenIndex, depth
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l175
							}
							position++
							goto l174
						l175:
							position, tokenIndex, depth = position175, tokenIndex175, depth175
						}
					}
				l172:
					depth--
					add(RulePegText, position171)
				}
				{
					add(RuleAction19, position)
				}
				depth--
				add(Rulefuzzy_param, position170)
			}
			return true
		l169:
			position, tokenIndex, depth = position169, tokenIndex169, depth169
			return false
		},
		/* 29 string <- <('"' <(term (space term)*)> '"' Action20)> */
		func() bool {
			position177, tokenIndex177, depth177 := position, tokenIndex, depth
			{
				position178 := position
				depth++
				if buffer[position] != rune('"') {
					goto l177
				}
				position++
				{
					position179 := position
					depth++
					if !rules[Ruleterm]() {
						goto l177
					}
				l180:
					{
						position181, tokenIndex181, depth181 := position, tokenIndex, depth
						if !rules[Rulespace]() {
							goto l181
						}
						if !rules[Ruleterm]() {
							goto l181
						}
						goto l180
					l181:
						position, tokenIndex, depth = position181, tokenIndex181, depth181
					}
					depth--
					add(RulePegText, position179)
				}
				if buffer[position] != rune('"') {
					goto l177
				}
				position++
				{
					add(RuleAction20, position)
				}
				depth--
				add(Rulestring, position178)
			}
			return true
		l177:
			position, tokenIndex, depth = position177, tokenIndex177, depth177
			return false
		},
		/* 30 keyword <- <((&('N') ('N' 'O' 'T')) | (&('O') ('O' 'R')) | (&('A') ('A' 'N' 'D')))> */
		func() bool {
			position183, tokenIndex183, depth183 := position, tokenIndex, depth
			{
				position184 := position
				depth++
				{
					switch buffer[position] {
					case 'N':
						if buffer[position] != rune('N') {
							goto l183
						}
						position++
						if buffer[position] != rune('O') {
							goto l183
						}
						position++
						if buffer[position] != rune('T') {
							goto l183
						}
						position++
						break
					case 'O':
						if buffer[position] != rune('O') {
							goto l183
						}
						position++
						if buffer[position] != rune('R') {
							goto l183
						}
						position++
						break
					default:
						if buffer[position] != rune('A') {
							goto l183
						}
						position++
						if buffer[position] != rune('N') {
							goto l183
						}
						position++
						if buffer[position] != rune('D') {
							goto l183
						}
						position++
						break
					}
				}

				depth--
				add(Rulekeyword, position184)
			}
			return true
		l183:
			position, tokenIndex, depth = position183, tokenIndex183, depth183
			return false
		},
		/* 31 valid_letter <- <(start_letter+ ((&('\\') ('\\' special_char)) | (&('-') '-') | (&('@') '@') | (&('.') '.') | (&('_') '_') | (&('?') '?') | (&('*') '*') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))*)> */
		func() bool {
			position186, tokenIndex186, depth186 := position, tokenIndex, depth
			{
				position187 := position
				depth++
				{
					position190 := position
					depth++
					{
						switch buffer[position] {
						case '\\':
							if buffer[position] != rune('\\') {
								goto l186
							}
							position++
							if !rules[Rulespecial_char]() {
								goto l186
							}
							break
						case '*':
							if buffer[position] != rune('*') {
								goto l186
							}
							position++
							break
						case '_':
							if buffer[position] != rune('_') {
								goto l186
							}
							position++
							break
						case '.':
							if buffer[position] != rune('.') {
								goto l186
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l186
							}
							position++
							break
						case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l186
							}
							position++
							break
						//default:
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l186
							}
							position++
							break
						default:
							if c := buffer[position]; !unicode.IsLetter(c) && !unicode.IsNumber(c) {
								goto l186
							}
							position++
							break
						}
					}

					depth--
					add(Rulestart_letter, position190)
				}
			l188:
				{
					position189, tokenIndex189, depth189 := position, tokenIndex, depth
					{
						position192 := position
						depth++
						{
							switch buffer[position] {
							case '\\':
								if buffer[position] != rune('\\') {
									goto l189
								}
								position++
								if !rules[Rulespecial_char]() {
									goto l189
								}
								break
							case '*':
								if buffer[position] != rune('*') {
									goto l189
								}
								position++
								break
							case '_':
								if buffer[position] != rune('_') {
									goto l189
								}
								position++
								break
							case '.':
								if buffer[position] != rune('.') {
									goto l189
								}
								position++
								break
							case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
								if c := buffer[position]; c < rune('0') || c > rune('9') {
									goto l189
								}
								position++
								break
							case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
								if c := buffer[position]; c < rune('a') || c > rune('z') {
									goto l189
								}
								position++
								break
							case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l189
							}
							position++
							break
						default:
							if c := buffer[position]; !unicode.IsLetter(c) && !unicode.IsNumber(c) {
								goto l189
							}
							position++
							break
							}
						}

						depth--
						add(Rulestart_letter, position192)
					}
					goto l188
				l189:
					position, tokenIndex, depth = position189, tokenIndex189, depth189
				}
			l194:
				{
					position195, tokenIndex195, depth195 := position, tokenIndex, depth
					{
						switch buffer[position] {
						case '\\':
							if buffer[position] != rune('\\') {
								goto l195
							}
							position++
							if !rules[Rulespecial_char]() {
								goto l195
							}
							break
						case '-':
							if buffer[position] != rune('-') {
								goto l195
							}
							position++
							break
						case '@':
							if buffer[position] != rune('@') {
								goto l195
							}
							position++
							break
						case '.':
							if buffer[position] != rune('.') {
								goto l195
							}
							position++
							break
						case '_':
							if buffer[position] != rune('_') {
								goto l195
							}
							position++
							break
						case '?':
							if buffer[position] != rune('?') {
								goto l195
							}
							position++
							break
						case '*':
							if buffer[position] != rune('*') {
								goto l195
							}
							position++
							break
						case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
							if c := buffer[position]; c < rune('0') || c > rune('9') {
								goto l195
							}
							position++
							break
						case 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':
							if c := buffer[position]; c < rune('a') || c > rune('z') {
								goto l195
							}
							position++
							break
						case 'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z':
							if c := buffer[position]; c < rune('A') || c > rune('Z') {
								goto l195
							}
							position++
							break
						default:
							if c := buffer[position]; !unicode.IsLetter(c) && !unicode.IsNumber(c) {
								goto l195
							}
							position++
							break
						}
					}

					goto l194
				l195:
					position, tokenIndex, depth = position195, tokenIndex195, depth195
				}
				depth--
				add(Rulevalid_letter, position187)
			}
			return true
		l186:
			position, tokenIndex, depth = position186, tokenIndex186, depth186
			return false
		},
		/* 32 start_letter <- <((&('\\') ('\\' special_char)) | (&('*') '*') | (&('_') '_') | (&('.') '.') | (&('0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9') [0-9]) | (&('a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h' | 'i' | 'j' | 'k' | 'l' | 'm' | 'n' | 'o' | 'p' | 'q' | 'r' | 's' | 't' | 'u' | 'v' | 'w' | 'x' | 'y' | 'z') [a-z]) | (&('A' | 'B' | 'C' | 'D' | 'E' | 'F' | 'G' | 'H' | 'I' | 'J' | 'K' | 'L' | 'M' | 'N' | 'O' | 'P' | 'Q' | 'R' | 'S' | 'T' | 'U' | 'V' | 'W' | 'X' | 'Y' | 'Z') [A-Z]))> */
		nil,
		/* 33 end_letter <- <([A-Z] / [a-z] / [0-9] / '*' / '?' / '_' / '.' / ('\\' special_char))> */
		nil,
		/* 34 special_char <- <((&(':') ':') | (&('\\') '\\') | (&('?') '?') | (&('*') '*') | (&('~') '~') | (&('"') '"') | (&('^') '^') | (&(']') ']') | (&('[') '[') | (&('}') '}') | (&('{') '{') | (&(')') ')') | (&('(') '(') | (&('!') '!') | (&('|') '|') | (&('&') '&') | (&('+') '+') | (&('-') '-'))> */
		func() bool {
			position199, tokenIndex199, depth199 := position, tokenIndex, depth
			{
				position200 := position
				depth++
				{
					switch buffer[position] {
					case ':':
						if buffer[position] != rune(':') {
							goto l199
						}
						position++
						break
					case '\\':
						if buffer[position] != rune('\\') {
							goto l199
						}
						position++
						break
					case '?':
						if buffer[position] != rune('?') {
							goto l199
						}
						position++
						break
					case '*':
						if buffer[position] != rune('*') {
							goto l199
						}
						position++
						break
					case '~':
						if buffer[position] != rune('~') {
							goto l199
						}
						position++
						break
					case '"':
						if buffer[position] != rune('"') {
							goto l199
						}
						position++
						break
					case '^':
						if buffer[position] != rune('^') {
							goto l199
						}
						position++
						break
					case ']':
						if buffer[position] != rune(']') {
							goto l199
						}
						position++
						break
					case '[':
						if buffer[position] != rune('[') {
							goto l199
						}
						position++
						break
					case '}':
						if buffer[position] != rune('}') {
							goto l199
						}
						position++
						break
					case '{':
						if buffer[position] != rune('{') {
							goto l199
						}
						position++
						break
					case ')':
						if buffer[position] != rune(')') {
							goto l199
						}
						position++
						break
					case '(':
						if buffer[position] != rune('(') {
							goto l199
						}
						position++
						break
					case '!':
						if buffer[position] != rune('!') {
							goto l199
						}
						position++
						break
					case '|':
						if buffer[position] != rune('|') {
							goto l199
						}
						position++
						break
					case '&':
						if buffer[position] != rune('&') {
							goto l199
						}
						position++
						break
					case '+':
						if buffer[position] != rune('+') {
							goto l199
						}
						position++
						break
					default:
						if buffer[position] != rune('-') {
							goto l199
						}
						position++
						break
					}
				}

				depth--
				add(Rulespecial_char, position200)
			}
			return true
		l199:
			position, tokenIndex, depth = position199, tokenIndex199, depth199
			return false
		},
		/* 35 open_paren <- <'('> */
		nil,
		/* 36 close_paren <- <')'> */
		nil,
		/* 37 open_incl <- <'['> */
		nil,
		/* 38 close_incl <- <']'> */
		nil,
		/* 39 open_excl <- <'{'> */
		nil,
		/* 40 close_excl <- <'}'> */
		nil,
		/* 41 space <- <((&('\r') '\r') | (&('\n') '\n') | (&('\t') '\t') | (&(' ') ' '))+> */
		func() bool {
			position208, tokenIndex208, depth208 := position, tokenIndex, depth
			{
				position209 := position
				depth++
				{
					switch buffer[position] {
					case '\r':
						if buffer[position] != rune('\r') {
							goto l208
						}
						position++
						break
					case '\n':
						if buffer[position] != rune('\n') {
							goto l208
						}
						position++
						break
					case '\t':
						if buffer[position] != rune('\t') {
							goto l208
						}
						position++
						break
					default:
						if buffer[position] != rune(' ') {
							goto l208
						}
						position++
						break
					}
				}

			l210:
				{
					position211, tokenIndex211, depth211 := position, tokenIndex, depth
					{
						switch buffer[position] {
						case '\r':
							if buffer[position] != rune('\r') {
								goto l211
							}
							position++
							break
						case '\n':
							if buffer[position] != rune('\n') {
								goto l211
							}
							position++
							break
						case '\t':
							if buffer[position] != rune('\t') {
								goto l211
							}
							position++
							break
						default:
							if buffer[position] != rune(' ') {
								goto l211
							}
							position++
							break
						}
					}

					goto l210
				l211:
					position, tokenIndex, depth = position211, tokenIndex211, depth211
				}
				depth--
				add(Rulespace, position209)
			}
			return true
		l208:
			position, tokenIndex, depth = position208, tokenIndex208, depth208
			return false
		},
		nil,
		/* 44 Action0 <- <{ p.AddTerm(buffer[begin:end]) }> */
		nil,
		/* 45 Action1 <- <{ p.StartBasic() }> */
		nil,
		/* 46 Action2 <- <{ p.StartGrouped() }> */
		nil,
		/* 47 Action3 <- <{ p.SetCompleted() }> */
		nil,
		/* 48 Action4 <- <{ p.StartRange(true) }> */
		nil,
		/* 49 Action5 <- <{ p.StartRange(false) }> */
		nil,
		/* 50 Action6 <- <{ p.AddField(buffer[begin:end]) }> */
		nil,
		/* 51 Action7 <- <{ p.AddRange(buffer[begin:end]) }> */
		nil,
		/* 52 Action8 <- <{ p.StartSubQuery() }> */
		nil,
		/* 53 Action9 <- <{ p.EndSubQuery() }> */
		nil,
		/* 54 Action10 <- <{ p.StartBasic() }> */
		nil,
		/* 55 Action11 <- <{ p.AddOp(OpBinOr) }> */
		nil,
		/* 56 Action12 <- <{ p.AddOp(OpBinAnd) }> */
		nil,
		/* 57 Action13 <- <{ p.AddTermOp(OpUnaryNot) }> */
		nil,
		/* 58 Action14 <- <{ p.AddTermOp(OpUnaryNot) }> */
		nil,
		/* 59 Action15 <- <{ p.AddTermOp(OpUnaryReq) }> */
		nil,
		/* 60 Action16 <- <{ p.AddTermOp(OpUnaryPro) }> */
		nil,
		/* 61 Action17 <- <{ p.AddOp(OpBoost) }> */
		nil,
		/* 62 Action18 <- <{ p.AddOp(OpFuzzy) }> */
		nil,
		/* 63 Action19 <- <{ p.AddTerm(buffer[begin:end]) }> */
		nil,
		/* 64 Action20 <- <{ p.AddTerm(buffer[begin:end]) }> */
		nil,
	}
	p.rules = rules
}
