// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import (
	"errors"
	"reflect"
)

// factory[] is a map that stores type constructors.
var factory map[string]func() interface{}

// functions[] is a map for storing functions with a suitable signature so that
// they can be called from within templates.
var functions map[string]func(g *Graph, p *Graph, i int) []byte // interface{}

// FunctionAddConstructor adds a factory kind of function to the context.
func FunctionAddConstructor(s string, f func() interface{}) {
	factory[s] = f
}

// FunctionAdd adds a function to the context.
func FunctionAdd(s string, f func(*Graph, *Graph, int) []byte) {
	functions[s] = f
}

// Function enables calling Go functions from templates. Path in templates
// are translated into Go functions if !type definitions are present.
//
// Functions and type methods are handled here, based on the two maps, factory[]
// and functions[].
//
// Also remote functions are called from here. A remote function is a call to
// a TCP/IP server, in which both the request and the response are binary encoded
// OGDL objects.
//
// (This code can be much improved)
func (g *Graph) Function(p *Graph, ix int, context *Graph) (interface{}, error) {

	n := g.Node("!type")

	if n == nil || n.Len() == 0 {
		return nil, nil
	}

	name := n.GetAt(0).String()

	// Case 1: simple function
	//
	// If type == "function", then call a function directly from the
	// functions[] table, no need to instantiate an object.

	if "function" == name {

		funame := p.GetAt(ix - 1).String()

		fu := functions[funame]
		if fu == nil {
			return nil, errors.New("function not in table " + funame)
		}

		arg := NilGraph()
		args := p.Out[ix]

		for i := 0; i < args.Len(); i++ {
			v := context.Eval(args.Out[i])

			arg.Add(_string(v))
		}

		return fu(context, arg, 0), nil
	}

	// Case 2: remote function

	if "rfunction" == name {

		var rf *RFunction
		var err error

		if n.Len() == 1 {
			rf, err = NewRFunction(g.Node("!init"))
			if err != nil {
				return nil, err
			}
			n.Add(rf)
		} else {
			rf = n.GetAt(1).This.(*RFunction)
		}

		arg := NewGraph(p.Out[ix].String())
		args := p.Out[ix+1]

		for _, a := range args.Out {
			v := context.Eval(a)

			g, ok := v.(*Graph);
			
			if ok && g.Len()>0 {
				arg.Add("_").Add(g)
			} else {
				arg.Add(v)
			}
		}
		return rf.Call(arg)
	}

	// Case 3: object with methods to be discovered through reflection

	var v reflect.Value

	// If !type has a second node, that means that it has been instantiated
	// already. The second node points to the type's instance.

	if n.Len() == 1 {

		// !type has one node, so instantiate.

		ff := factory[name]
		if ff == nil {
			return nil, errors.New("function not in table " + name)
		}

		itf := ff()
		v = reflect.ValueOf(itf)

		// Add the object as second node of !type. Next time w'll pick this object.
		n.Add(v)

		// If !init is defined, the init(Graph) function is called on the instantiated type.
		nn := g.Node("!init")

		if nn != nil {
			v.MethodByName("Init").Call([]reflect.Value{reflect.ValueOf(nn)})
		}
	} else {
		v = n.GetAt(1).This.(reflect.Value)
	}

	// exec: as per path
	//
	// p[i] is a function or field name
	// rest are arguments.

	// obj = p.GetAt(i)

	fn := p.GetAt(ix)
	ag := p.GetAt(ix + 1)

	if "!g" != ag.String() {
		return nil, errors.New("missing !g")
	}

	if fn == nil {
		s := "No method " + fn.Text()
		return s, errors.New(s)
	}

	fname := fn.String()

	// TODO: Check if it is a field

	// Check if it is a method
	me := v.MethodByName(fname)

	if !me.IsValid() {
		s := "No method " + fname
		return s, errors.New(s)
	}

	// Build arguments in the form []reflect.Value

	var args []reflect.Value

	for _, arg := range ag.Out {
		a := context.Eval(arg)
		args = append(args, reflect.ValueOf(a))
	}

	return me.Call(args)[0].Interface(), nil
}

// Function2 enables calling Go functions from templates. 
//
func (g *Graph) Function2 (p *Graph, ix int, context *Graph) (interface{}, error) {

	// g.This must be an object with associated fields or methods

    // XXX reject as early as possible any non-struct here.
    if g.Len()!=1 {
        return nil,nil
    }

// println("type:",reflect.TypeOf(g.GetAt(0).This).String(),"->",g.String(),g.GetAt(0).String())

    v := reflect.ValueOf(g.GetAt(0).This)
    if ! v.IsValid() {
        return nil, nil
    }
    
	fn := p.GetAt(ix)
	ag := p.GetAt(ix + 1)

	if fn == nil {
		s := "No method " + fn.Text()
		return s, errors.New(s)
	}

	fname := fn.String()

	// Check if it is a method
	me := v.MethodByName(fname)

	if !me.IsValid() {
	    // Try field
	    if v.Kind()==reflect.Struct {
	        v = v.FieldByName(fname)
	        if v.IsValid() {
	            return v.Interface(), nil
	        }
	    }
	    
		s := "No method or field " + fname
		return s, errors.New(s)
	}

	// Build arguments in the form []reflect.Value

	var args []reflect.Value

	for _, arg := range ag.Out {
		a := context.Eval(arg)		
		args = append(args, reflect.ValueOf(a))
	}

	return me.Call(args)[0].Interface(), nil
}

func init() {
	factory = make(map[string]func() interface{})
	factory["nil"] = nilGraphI

	functions = make(map[string]func(g *Graph, p *Graph, i int) []byte)
	functions["T"] = templateProcess
}

// Example functions and objects

func templateProcess(context *Graph, p *Graph, i int) []byte {
	t := NewTemplate(p.Text())
	return t.Process(context)
}

func nilGraphI() interface{} {
	return NilGraph()
}
