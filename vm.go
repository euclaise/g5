package main

import (
	"fmt"
	"math/big"
	"os"
)

var OutputPortStack = []OutputPort{{os.Stdout}}
var InputPortStack = []InputPort{{os.Stdin}}

type Op uint8

const (
	Imm Op = iota
	GetVar
	Call
	Lambda
	Set
	Define
	If
	SaveScope
)

type Ins struct {
	op    Op
	imm   Value
	nargs int
}

func (scope *Scope) Lookup(v Value) *Scope {
	sym, ok := v.(Symbol)
	if !ok {
		_, ok = v.(Scoped)
		if !ok {
			fmt.Printf("Found type: %T\n", v)
			panic("Tried to lookup non-symbol")
		}
		sym = v.(Scoped).Symbol
		scope = scope.Lookup(v.(Scoped).Scope)
	}

	if scope == nil {
		return nil
	}

	if _, ok := scope.m[sym]; !ok {
		return scope.super.Lookup(sym)
	}
	return scope
}

func (p *Procedure) Eval() error {
begin:
	for len(p.Ins) > 0 {
		ins := p.Ins[0]
		p.Ins = p.Ins[1:]

		switch ins.op {
		case Imm:
			stack.Push(ins.imm)
		case GetVar:
			var scope *Scope
			switch ins.imm.(type) {
			case Symbol, Scoped:
				scope = p.Scope.Lookup(ins.imm)
			default:
				return fmt.Errorf("Tried to look up non-variable")
			}
			if scope == nil {
				return fmt.Errorf("Could not find variable: %s",
					SymbolNames[ins.imm.(Symbol)])
			}

			sym, ok := ins.imm.(Symbol)
			if !ok {
				sym = ins.imm.(Scoped).Symbol
			}
			stack.Push(scope.m[sym])
		case Call:
			callee := stack.Pop()
			newp_template, ok := callee.(*Procedure)
			if !ok {
				fmt.Println("CALL:")
				PrintValue(callee)
				fmt.Println()
				return fmt.Errorf("Call to non-procedure (%T)", callee)
			}

			var nargs int
			if ins.nargs == -1 { // (call-with-values ...) will use nargs of -1
				nargs_bi := big.Int(stack.Pop().(Integer))
				nargs = int(nargs_bi.Int64())
			} else {
				nargs = ins.nargs
			}

			var newp *Procedure
			if newp_template.IsCont {
				newp = newp_template
			} else {
				newp = &Procedure{}
				*newp = *newp_template
			}

			if newp.CallCC != nil {
				if res := newp.CallCC(p, nargs); res != nil {
					return res
				}
			} else if newp.Builtin != nil {
				if res := newp.Builtin(nargs); res != nil {
					return res
				}
			} else {
				if newp.IsCont {
					top := stack.Top()
					stack = *newp.StackRest
					stack.Push(top)
					newp.IsCont = false
				} else {
					newp.Scope.m = map[Symbol]Value{}
					n := nargs
					cur := newp.Args
					_, ispair := newp.Args.(*Pair)
					for n > 0 && ispair {
						if cur == Empty {
							return fmt.Errorf("Wrong arg count (got %d)", n)
						}
						sym, ok := (*cur.(*Pair).Car).(Symbol)
						if !ok {
							panic("Non-symbol argument?")
						}
						newp.Scope.m[sym] = stack.Pop()
						n--
						cur = *cur.(*Pair).Cdr
						if _, ok := cur.(*Pair); !ok {
							break
						}
					}

					// Dot arg
					if s, ok := cur.(Symbol); ok {
						rest := &Pair{}
						cur := rest
						if n == 0 {
							newp.Scope.m[s] = Empty
						} else {
							for n > 0 {
								v := stack.Pop()
								n--
								cur.Car = &v

								if n == 0 {
									cdr := Empty
									cur.Cdr = &cdr
									break
								}
								var next Value = &Pair{}
								cur.Cdr = &next
								cur = next.(*Pair)
							}
							newp.Scope.m[s] = rest
						}
					}
				}

				if len(p.Ins) == 0 { // Tail call
					p = newp
					goto begin
				}

				stack_pos := len(stack)
				res := newp.Eval()
				// Clear temps from stack
				top := stack.Top()
				stack = append(stack[:stack_pos], top)
				if res != nil {
					return res
				}
			}
		case Lambda: // Procedure -> *Procedure
			lambda := ins.imm.(Procedure)
			lambda.Scope = Scope{}
			lambda.Scope.m = map[Symbol]Value{}

			scope := p.Scope
			lambda.Scope.super = &scope

			stack.Push(Value(&lambda))
		case Set:
			sym := Unscope(ins.imm).(Symbol)
			scope := p.Scope.Lookup(ins.imm)
			if scope == nil {
				scope = &p.Scope
			}
			scope.m[sym] = stack.Top()
		case Define:
			sym := ins.imm.(Symbol)
			if _, ok := p.Scope.m[sym]; ok {
				fmt.Printf("WARNING: Redefining binding %s\n", SymbolNames[sym])
			}
			p.Scope.m[sym] = stack.Top()
		case If:
			cond, isbool := stack.Pop().(Boolean)
			lt := stack.Pop().(Procedure)
			lf := Procedure{}

			if ins.nargs == 3 {
				lf = stack.Pop().(Procedure)
			}

			var res error
			if !isbool || bool(cond) {
				lt.Scope = p.Scope
				res = lt.Eval()
			}

			if ins.nargs == 3 {
				lf.Scope = p.Scope
				if isbool && !bool(cond) {
					res = lf.Eval()
				}
			}
			if res != nil {
				return res
			}
		case SaveScope:
			stack.Push(&p.Scope)
		}
	}
	return nil
}

func (ins Ins) Print() {
	switch ins.op {
	case Imm:
		fmt.Print("IMM")
		fmt.Print("[")
		PrintValue(ins.imm)
		fmt.Println("]")
	case GetVar:
		fmt.Print("GETVAR")
		fmt.Print("[")
		PrintValue(ins.imm)
		fmt.Println("]")
	case Call:
		fmt.Print("CALL")
		fmt.Printf("(%d)\n", ins.nargs)
	case Set:
		fmt.Print("SET!")
		fmt.Print("[")
		PrintValue(ins.imm)
		fmt.Println("]")
	case Define:
		fmt.Print("DEFINE")
		fmt.Print("[")
		PrintValue(ins.imm)
		fmt.Println("]")
	case Lambda:
		fmt.Print("LAMBDA ")
		PrintValue(ins.imm.(Procedure).Args)
		fmt.Println()
	case If:
		fmt.Println("IF")
	case SaveScope:
		fmt.Println("SAVE-SCOPE")
	default:
		fmt.Println("[unknown]")
	}
}
