// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"bytes"
)

// NewTemplate parses a text template given as a string and converts it to a Graph.
// Templates have fixed and variable parts. Variables all begin with '$'.
//
// A template is a text file in any format: plain text, HTML, XML, OGDL or
// whatever. The dolar sign acts as an escape character that switches from the
// text to the variable plane. Parsed templates are converted back to text
// by evaluating the variable parts against a Graph object, by means of the
// Process() method.
//
// Template grammar
//
//     template ::= ( text | variable )*
//
//     variable ::= ('$' path) | ('$' '(' expression ')') | ('$' '{' expression '}')
//     path ::= as defined in path.go
//     expression ::= as defined in expression.go
//
// Some variables act as directives: $if, $else, $end, $for, $break.
//
//    $if(expression)
//    $else
//    $end
//
//    $for(destPath,sourcepath)
//      $break
//    $end
//
func NewTemplate(s string) *Graph {
	p := NewStringParser(s)
	p.Template()

	t := p.GraphTop(TypeTemplate)
	t.Ast()
	t.simplify()
	t.flow()

	return t
}

// Process processes the parsed template, returning the resulting text in a byte array.
// The variable parts are resolved out of the Graph given.
func (t *Graph) Process(c *Graph) []byte {

	buffer := &bytes.Buffer{}

	t.process(c, buffer)

	return buffer.Bytes()
}

func (t *Graph) process(c *Graph, buffer *bytes.Buffer) bool {

	falseIf := false

	for _, n := range t.Out {
		s := n.String()

		switch s {
		case TypePath:
			i := c.Eval(n)

			// If i is a graph, we want the full graph converted to string,
			// not just the root node (which is what _string() returns.

			if g, ok := i.(*Graph); ok {
				buffer.WriteString(g.Text())
			} else {
				buffer.WriteString(_string(c.Eval(n)))
			}
		case TypeExpression:
			// Silent evaluation
			c.Eval(n)
		case TypeIf:
			// evaluate the expression
			b := c.EvalBool(n.GetAt(0).GetAt(0))

			if b {
				n.GetAt(1).process(c, buffer)
				falseIf = false
			} else {
				falseIf = true
			}
		case TypeElse:
			// if there was a previous if evaluating to false:
			if falseIf {
				n.process(c, buffer)
				falseIf = false
			}
		case TypeFor:
			// The first subnode (of !g) is a path
			// The second is an expression evaluating to a list of elements
			i := c.Eval(n.GetAt(0).GetAt(1))

			// Check that i is iterable

            gi, ok := i.(*Graph);
			if !ok || gi==nil {
				return true
			}
						
			// The third is the subtemplate to travel
			// println ("for type: ",reflect.TypeOf(i).String(), "ok",ok)
			// Assing expression value to path
			// XXX if not Graph
			for _, ee := range gi.Out {
				c.assign(n.GetAt(0).GetAt(0).GetAt(0), ee, '=')
				brk := n.GetAt(1).process(c, buffer)
				if brk {
					break
				}
			}
		case TypeBreak:
			return true

		default:
			buffer.WriteString(n.String())
		}
	}
	return false
}

// simplify converts !p TYPE in !TYPE for keywords if, end, else for and break.
func (t *Graph) simplify() {
	for _, node := range t.Out {
		if TypePath == node.String() {
			s := node.GetAt(0).String()

			switch s {
			case "if":
				node.This = TypeIf
				node.DeleteAt(0)
			case "end":
				node.This = TypeEnd
				node.DeleteAt(0)
			case "else":
				node.This = TypeElse
				node.DeleteAt(0)
			case "for":
				node.This = TypeFor
				node.DeleteAt(0)
			case "break":
				node.This = TypeBreak
				node.DeleteAt(0)
			}
		}
	}

}

// flow nests 'if' and 'for' loops.
func (t *Graph) flow() {
	n := 0
	var nod *Graph

	for i := 0; i < t.Len(); i++ {

		node := t.Out[i]
		s := node.String()

		if s == TypeIf || s == TypeFor {
			n++
			if n == 1 {
				nod = node.Add(TypeTemplate)
				continue
			}
		}

		if s == TypeElse {
			if n == 1 {
				nod.flow()
				nod = node
				continue
			}
		}

		if s == TypeEnd {
			n--
			if n == 0 {
				nod.flow()
				t.DeleteAt(i)
				i--
				continue
			}
		}

		if n > 0 {
			nod.Add(node)
			t.DeleteAt(i)
			i--
		}
	}

}
