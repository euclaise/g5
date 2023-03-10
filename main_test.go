package main

import (
	"math/big"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	Top.Scope = TopScope // Put builtins into top-level scope

	if int(SymLast) != len(SymbolNames) {
		panic("Symbol table length mismatch")
	}

	Top.Run(Init, true)
	Top.Run(CaseLambdaSRFI, true)
	Top.Run(ListsSRFI, true)

	for k, v := range TopScope.m { // Copy unmodified scope into basescope
		BaseScope[k] = v
	}
	os.Exit(m.Run())
}

func TestLambdas(t *testing.T) {
	Top.Run("(define add (lambda (x y) (+ x y)))", true)
	result := stack.Top()
	if _, ok := result.(*Procedure); !ok {
		t.Errorf("Expected procedure, got %T", result)
	}

	Top.Run("(add 2 3)", true)
	result, ok := stack.Top().(Integer)
	if !ok {
		t.Errorf("Expected integer, got %T", result)
	}

	if result := big.Int(result.(Integer)); result.Cmp(big.NewInt(5)) != 0 {
		t.Errorf("Expected integer, got %v", result.String())
	}
}

func TestAdder(t *testing.T) {
	Top.Run("(define make-adder (lambda (x) (lambda (y) (+ x y))))", true)
	result := stack.Top()
	if _, ok := result.(*Procedure); !ok {
		t.Errorf("Expected procedure, got %T", result)
	}

	Top.Run("(define add-2 (make-adder 2))", true)
	result = stack.Top()
	if _, ok := result.(*Procedure); !ok {
		t.Errorf("Expected procedure, got %T", result)
	}

	Top.Run("(add-2 3)", true)
	result, ok := stack.Top().(Integer)
	if !ok {
		t.Errorf("Expected integer, got %T", result)
	}

	if result := big.Int(result.(Integer)); result.Cmp(big.NewInt(5)) != 0 {
		t.Errorf("Expected integer, got %v", result.String())
	}
}

func TestCounter(t *testing.T) {
	Top.Run("(define (make-ctr) (set! count 0)"+
		"(lambda (ctr) (set! count (+ count 1)) count))", true)
	result := stack.Top()
	if _, ok := result.(*Procedure); !ok {
		t.Errorf("Expected procedure, got %T", result)
	}

	Top.Run("(define ctr (make-ctr))", true)
	result = stack.Top()
	if _, ok := result.(*Procedure); !ok {
		t.Errorf("Expected procedure, got %T", result)
	}

	Top.Run("(ctr)", true)
	result, ok := stack.Top().(Integer)
	if !ok {
		t.Errorf("Expected integer, got %T", result)
	}

	if result := big.Int(result.(Integer)); result.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Expected 1, got %v", result.String())
	}

	Top.Run("(ctr)", true)
	result, ok = stack.Top().(Integer)
	if !ok {
		t.Errorf("Expected integer, got %T", result)
	}

	if result := big.Int(result.(Integer)); result.Cmp(big.NewInt(2)) != 0 {
		t.Errorf("Expected 2, got %v", result.String())
	}
}

func TestLet(t *testing.T) {
	Top.Run("(let ((a 0) (b 1)) b)", true)
	result, ok := stack.Top().(Integer)
	if !ok {
		t.Errorf("Expected integer, got %T", result)
	}

	if result := big.Int(result); result.Cmp(big.NewInt(1)) != 0 {
		t.Errorf("Expected integer, got %v", result.String())
	}
}

func TestAnd(t *testing.T) {
	Top.Run("(and #t #t)", true)
	result, ok := stack.Top().(Boolean)
	if !ok {
		t.Errorf("Expected boolean, got %T", result)
	}

	if result != true {
		t.Errorf("Expected #t, got #f")
	}
}

func TestOr(t *testing.T) {
	Top.Run("(or #t #f)", true)
	result, ok := stack.Top().(Boolean)
	if !ok {
		t.Errorf("Expected boolean, got %T", result)
	}

	if result != true {
		t.Errorf("Expected true, got false")
	}
}

func TestLetrec(t *testing.T) {
	Top.Run("(letrec ((a #t)) #t)", true)
	result, ok := stack.Top().(Boolean)
	if !ok {
		t.Errorf("Expected boolean, got %T", result)
	}

	if result != true {
		t.Errorf("Expected true, got false")
	}
}
