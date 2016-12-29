package main

import (
	"bufio"
	"bytes"
	"io"
)

type Treat struct {
	Key      Key
	Operator Operator
	Value    string
}

type Statement struct {
	Source  io.Reader
	Phrases []string
	Treats  []Treat
}

/* Keys are the treat identifiers. Only keys specified within the keys
 * map[string]Key are identified as being part of a treat! Otherwise search
 * queries like "csi:miami" would not be possible.
 */
type Key int

const (
	keyExtension Key = iota
	keySize
	keyServer
	keyType
)

var keys = map[string]Key{
	"extension": keyExtension,
	"size":      keySize,
	"type":      keyType,
}

/* Operators specify how the treat should be applied. Only operators specified
 * within the operators map[string]Operator are identified as being part of
 * a treat! Multiple characters are possible, e.g. "=="
 */
type Operator int

const (
	EQUALS Operator = iota
	NOT
	LTE
	GTE
)

var operators = map[string]Operator{
	":": EQUALS,
	"!": NOT,
	"<": LTE,
	">": GTE,
}

/* LEXER
 * Tokenizes the input for further processing. Usually you just skip this and
 * take a look at the parser.
 */
type Token int

const (
	NIL Token = iota
	KEY
	OPERATOR
	STRING
)

const EOF = rune(0)

type TreatLexer struct {
	r *bufio.Reader
}

func CreateTreatLexer(src io.Reader) (tl TreatLexer) {
	tl = TreatLexer{
		r: bufio.NewReader(src),
	}
	return
}

/* Get the next token from a query string and the corresponding value. Parsing
 * has to be done elsewhere - this is just the lexer.
 */
func (tl *TreatLexer) Next() (token Token, value string) {
	token = NIL
	var current bytes.Buffer

Loop:
	for {
		ch, _, err := tl.r.ReadRune()

		// No character or EOF
		if err != nil || ch == EOF {
			break Loop
		}

		// Whitespaces separate our tokens
		if ch == ' ' || ch == '\t' || ch == '\n' {
			// If we found nothing by now, better continue until we have something
			if token == NIL {
				continue
			}

			// Otherwise this marks the end of our token
			break Loop
		}

		current.WriteRune(ch)
		_, isKey := keys[current.String()]
		_, isOperator := operators[current.String()]

		switch {
		// Find keys
		case isKey:
			token = KEY
			break Loop
		/* Find operators. Operators always follow a Key as otherwise the token
		 * is not splitted.
		 */
		case isOperator:
			token = OPERATOR
			break Loop
		// Fuck it, it's an arbitary string
		default:
			token = STRING
		}
	}

	value = current.String()
	return
}

/* PARSER
 * Tries to recognize patterns within the already tokenized input via a state
 * machine so we get out a Statement consisting of Phrases and Treats.
 */
type ParserState int

const (
	PKEY ParserState = iota
	POPERATOR
	PSTRING
)

/* Parses a query string which might contain treat directives into a Statement
 * struct. Treats consist of [key][operator][value], e.g. size>20mb.
 * Everything not being a treat consisting of the pre-specified keys and
 * operators is interpreted as part of the actual search query.
 *
 * Example:
 *   fp := TreatParser("filetype:pdf size>20mb size<50mb server!ftp.gnu.org scientific paper writing")
 */
func TreatParser(src io.Reader) (stmt Statement) {
	stmt = Statement{Source: src}

	state := PSTRING
	var previousValue string
	var currentTreat Treat

	lexer := CreateTreatLexer(src)
	for {
		token, value := lexer.Next()

		switch token {
		case KEY:
			currentTreat = Treat{
				Key: keys[value],
			}
			state = PKEY
			break
		case OPERATOR:
			state = POPERATOR
			currentTreat.Operator = operators[value]
			break
		case STRING:
			// Possible states at this point: PKEY, POPERATOR, PSTRING

			// We already have key+operator, add the value
			if state == POPERATOR {
				currentTreat.Value = value
				stmt.Treats = append(stmt.Treats, currentTreat)
				state = PSTRING
				break
			}

			// Whoops a key without a following operator. Think of both as phrases!
			if state == PKEY {
				stmt.Phrases = append(stmt.Phrases, previousValue)
			}

			// Otherwise: This is a usual string, just add it to the phrases …
			stmt.Phrases = append(stmt.Phrases, value)
			state = PSTRING
			break
		}

		previousValue = value

		if token == NIL {
			break
		}
	}

	return
}
