package main

import "fmt"

type Node interface {
	invalid()
	fmt.Formatter
	Equal(Node) bool
}

// Empty
var _ Node = &NodeEmpty{}

type NodeEmpty struct{}

func NewNodeEmpty() *NodeEmpty {
	return &NodeEmpty{}
}

func (NodeEmpty) invalid() {}

func (NodeEmpty) Format(st fmt.State, verb rune) {
	fmt.Fprint(st, "/* empty */")
}

func (NodeEmpty) Equal(v Node) bool {
	_, ok := v.(*NodeEmpty)
	return ok
}

// term ref
var _ Node = &NodeTerm{}

type NodeTerm struct {
	ref  string
	term bool
}

func NewNodeTerm(ref string, term bool) *NodeTerm {
	return &NodeTerm{ref, term}
}

func (NodeTerm) invalid() {}

func (s *NodeTerm) Format(st fmt.State, verb rune) {
	fmt.Fprint(st, s.ref)
}

func (s *NodeTerm) Equal(v Node) bool {
	rv, ok := v.(*NodeTerm)
	return ok && rv.ref == s.ref
}

// optional
var _ Node = &NodeOpt{}

type NodeOptType byte

const (
	NodeOptTypeOption NodeOptType = iota
	NodeOptTypeZeroMore
	NodeOptTypeOneMore
)

type NodeOpt struct {
	ref Node
	typ NodeOptType
}

func NewNodeOpt(ref Node, typ NodeOptType) *NodeOpt {
	return &NodeOpt{ref, typ}
}

func (NodeOpt) invalid() {}

func (s *NodeOpt) Format(st fmt.State, verb rune) {
	switch s.typ {
	case NodeOptTypeOneMore:
		fmt.Fprintf(st, "(")
		s.ref.Format(st, verb)
		fmt.Fprintf(st, ")+")
	case NodeOptTypeZeroMore:
		fmt.Fprintf(st, "(")
		s.ref.Format(st, verb)
		fmt.Fprintf(st, ")*")
	case NodeOptTypeOption:
		fmt.Fprintf(st, "(")
		s.ref.Format(st, verb)
		fmt.Fprintf(st, ")?")
	}
}

func (s *NodeOpt) Equal(v Node) bool {
	rv, ok := v.(*NodeOpt)
	return ok && rv.typ == s.typ && rv.ref.Equal(s.ref)
}

// seq
var _ Node = &NodeSeq{}

type NodeSeqType byte

const (
	NodeSeqTypeSequential NodeSeqType = iota
	NodeSeqTypeChoice
)

type NodeSeq struct {
	Nodes []Node
	typ   NodeSeqType
}

func NewNodeSeq(typ NodeSeqType) *NodeSeq {
	return &NodeSeq{typ: typ}
}

func (NodeSeq) invalid() {}

func (s *NodeSeq) IsType(t NodeSeqType) bool {
	return s.typ == t
}

func (s *NodeSeq) Format(st fmt.State, verb rune) {
	if verb == 'v' {
		var typ string
		switch s.typ {
		case NodeSeqTypeSequential:
			typ = "seq"
		case NodeSeqTypeChoice:
			typ = "choice"
		}
		fmt.Fprintf(st, "%s(", typ)
	}
	for i, n := range s.Nodes {
		if i > 0 {
			if verb == 'v' {
				fmt.Fprintf(st, ", ")
			} else {
				switch s.typ {
				case NodeSeqTypeSequential:
					fmt.Fprintf(st, " ")
				case NodeSeqTypeChoice:
					fmt.Fprintf(st, " | ")
				}
			}
		}
		n.Format(st, verb)
	}
	if verb == 'v' {
		fmt.Fprintf(st, ")")
	}
}

func (s *NodeSeq) Equal(v Node) bool {
	rv, ok := v.(*NodeSeq)
	ok = ok && rv.typ == s.typ && len(rv.Nodes) == len(s.Nodes)
	if !ok {
		return ok
	}
	for i := range s.Nodes {
		ok = ok && s.Nodes[i].Equal(rv.Nodes[i])
	}
	return ok
}
