package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"go/token"
	"io"
	"log"
	"os"
	"strings"

	"modernc.org/y"
)

var termName = map[string]string{
	"identifier":         "identifier",
	"singleAtIdentifier": "singleAtIdentifier",
	"doubleAtIdentifier": "doubleAtIdentifier",
	"invalid":            "invalid",
	"hintComment":        "hintComment",
	"stringLit":          "stringLit",
	"floatLit":           "floatLit",
	"decLit":             "decLit",
	"intLit":             "intLit",
	"hexLit":             "hexLit",
	"bitLit":             "bitLit",
}

type TermProc struct {
	sym     map[string]*y.Symbol
	bsym    map[string]Node
	visited map[string]struct{}
	passes  []func(Node) (Node, bool)
}

func NewTermProc(p *y.Parser) *TermProc {
	r := &TermProc{
		sym:     p.Syms,
		bsym:    make(map[string]Node),
		visited: make(map[string]struct{}),
	}
	r.passes = append(r.passes,
		r.passEmptyOrSingle,
		r.passChoiceOpt,
		r.passChoiceMerge,
		r.passTermInline,
	)

	return r
}

func (p *TermProc) toBNF(ref string) {
	refs := []string{ref}
	for i := 0; i < len(refs); i++ {
		sym := p.sym[refs[i]]
		if sym == nil {
			continue
		}
		if sym.IsTerminal {
			t := termName[sym.Name]
			if t != "" {
				p.bsym[sym.Name] = NewNodeTerm(t, true)
			} else if sym.LiteralString != "" {
				p.bsym[sym.Name] = NewNodeTerm(sym.LiteralString, true)
			} else {
				p.bsym[sym.Name] = NewNodeTerm(sym.Name, true)
			}
		} else {
			bsym := NewNodeSeq(NodeSeqTypeChoice)
			for _, rule := range sym.Rules {
				bsymc_rule := NewNodeSeq(NodeSeqTypeSequential)
				for _, comp := range rule.Components {
					bsymc_rule.Nodes = append(bsymc_rule.Nodes, NewNodeTerm(comp, false))
				}
				bsym.Nodes = append(bsym.Nodes, bsymc_rule)
			}
			p.bsym[sym.Name] = bsym
			p.visited[refs[i]] = struct{}{}
			for _, ref := range p.collectRef(bsym) {
				if _, ok := p.visited[ref]; !ok {
					p.visited[ref] = struct{}{}
					refs = append(refs, ref)
				}
			}
		}
	}
	for i := 0; i < len(refs); i++ {
		p.bsym[refs[i]], _ = p.rewrite(p.bsym[refs[i]])
	}
}

func (p *TermProc) rewrite(v Node) (Node, bool) {
	//fmt.Printf("#rewrite %+v\n", v)
	switch rv := v.(type) {
	case *NodeSeq:
		changed := false
		for {
			it_changed, pass_changed := false, false
			for i, n := range rv.Nodes {
				rv.Nodes[i], pass_changed = p.rewrite(n)
				it_changed = it_changed || pass_changed
			}
			for _, pass := range p.passes {
				v, pass_changed = pass(v)
				it_changed = it_changed || pass_changed
			}
			if !it_changed {
				break
			}
			changed = true
		}
		return v, changed
	default:
		changed := false
		for {
			it_changed, pass_changed := false, false
			for _, pass := range p.passes {
				v, pass_changed = pass(v)
				it_changed = it_changed || pass_changed
			}
			if !it_changed {
				break
			}
			changed = true
		}
		return v, changed
	}
}

func (p *TermProc) passChoiceOpt(v Node) (Node, bool) {
	rv, ok := v.(*NodeSeq)
	if !ok || !rv.IsType(NodeSeqTypeChoice) {
		return v, false
	}

	opt := -1
	for i, n := range rv.Nodes {
		if _, ok := n.(*NodeEmpty); ok {
			opt = i
			break
		}
	}
	if opt != -1 {
		rv.Nodes = append(rv.Nodes[:opt], rv.Nodes[opt+1:]...)
		return NewNodeOpt(rv, NodeOptTypeOption), true
	}

	return rv, false
}

func (p *TermProc) passEmptyOrSingle(v Node) (Node, bool) {
	rv, ok := v.(*NodeSeq)
	if !ok {
		return v, false
	}

	if len(rv.Nodes) == 0 {
		return NewNodeEmpty(), true
	} else if len(rv.Nodes) == 1 {
		return rv.Nodes[0], true
	}
	return rv, false
}

func (p *TermProc) isChoiceSeq(v Node) (*NodeSeq, bool) {
	rv, ok := v.(*NodeSeq)
	if !ok || !rv.IsType(NodeSeqTypeChoice) {
		return rv, false
	}
	opt := true
	for _, n := range rv.Nodes {
		m, ok := n.(*NodeSeq)
		opt = opt && ok && m.IsType(NodeSeqTypeSequential)
	}
	return rv, opt
}

func (p *TermProc) passChoiceMerge(v Node) (Node, bool) {
	rv, ok := p.isChoiceSeq(v)
	if !ok {
		return v, false
	}

	if len(rv.Nodes) < 2 {
		return rv, false
	}

	return rv, false

	/*
		var preNodes []Node
		var postNodes []Node
		for {
			preNodesLen, postNodesLen := len(preNodes), len(postNodes)

			seqNodes := rv.Nodes[0].(*NodeSeq).Nodes
			prefix, suffix := seqNodes[0], seqNodes[len(seqNodes)-1]
			samePrefix, sameSuffix := true, true
			for _, n := range rv.Nodes {
				m := n.(*NodeSeq)
				samePrefix = samePrefix && m.Nodes[0].Equal(prefix)
				sameSuffix = sameSuffix && m.Nodes[len(m.Nodes)-1].Equal(suffix)
			}
			if samePrefix {
				preNodes = append(preNodes, seqNodes[0])
				for _, n := range rv.Nodes {
					m := n.(*NodeSeq)
					m.Nodes = m.Nodes[1:]
				}
			}
			if sameSuffix {
				postNodes = append(postNodes, seqNodes[len(seqNodes)-1])
				for _, n := range rv.Nodes {
					m := n.(*NodeSeq)
					m.Nodes = m.Nodes[:len(m.Nodes)-1]
				}
			}

			npreNodesLen, npostNodesLen := len(preNodes), len(postNodes)
			if preNodesLen == npreNodesLen && postNodesLen == npostNodesLen {
				break
			}
		}
		if len(preNodes) == 0 && len(postNodes) == 0 {
			return rv, false
		}
		re := NewNodeSeq(NodeSeqTypeSequential)
		re.Nodes = append(re.Nodes, preNodes...)
		re.Nodes = append(re.Nodes, rv)
		re.Nodes = append(re.Nodes, postNodes...)
		return re, true
	*/
}

func (p *TermProc) passTermInline(v Node) (Node, bool) {
	rv, ok := v.(*NodeTerm)
	if !ok {
		return v, false
	}
	if rv.term {
		return rv, false
	}

	rov, ok := p.bsym[rv.ref].(*NodeTerm)
	if ok && rov.term {
		return rov, true
	}
	return rv, false
}

func (p *TermProc) collectRef(v Node) []string {
	var r []string
	switch rv := v.(type) {
	case *NodeEmpty:
	case *NodeTerm:
		r = append(r, rv.ref)
	case *NodeOpt:
		r = p.collectRef(rv.ref)
	case *NodeSeq:
		for _, n := range rv.Nodes {
			r = append(r, p.collectRef(n)...)
		}
	}
	return r
}

func main() {
	yaccFile := flag.String("in", "parser.y", "yacc file")
	cmdFile := flag.String("c", "-", "DSL scripts")
	flag.Parse()

	p, err := y.ProcessFile(token.NewFileSet(), *yaccFile, &y.Options{})
	if err != nil {
		log.Fatal(err)
	}

	var cmdB *bufio.Reader
	if *cmdFile == "-" {
		cmdB = bufio.NewReader(os.Stdin)
	} else {
		cmdF, err := os.Open(*cmdFile)
		if err != nil {
			log.Fatal(err)
		}
		defer cmdF.Close()
		cmdB = bufio.NewReader(cmdF)
	}

	tp := NewTermProc(p)
	for {
		line, err := cmdB.ReadString('\n')
		if errors.Is(err, io.EOF) {
			break
		}

		_spec2term := strings.Split(line[:len(line)-1], " ")
		if len(_spec2term) != 2 {
			log.Fatal("need a command specifier and a term name")
		}
		_, sym := _spec2term[0], _spec2term[1]

		tp.toBNF(sym)
		fmt.Printf("int main() {\n%+v;\n}", tp.bsym[sym])
		//fmt.Printf("%s;\n", tp.bsym[sym])
	}
}
