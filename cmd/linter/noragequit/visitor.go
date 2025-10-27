package noragequit

import (
	"fmt"
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const (
	selFatal  string = "log.Fatal"
	selFatalf string = "log.Fatalf"
	selExit   string = "os.Exit"
)

type RageQuitVisitor struct {
	excludedPackages map[string]bool

	nodeStack       []*ast.Node
	isInMainPackage bool
	isInMainFunc    bool

	issues []analysis.Diagnostic
}

func NewRageQuitVisitor(excludedPkgs string) *RageQuitVisitor {
	v := &RageQuitVisitor{
		excludedPackages: make(map[string]bool),
	}

	eps := strings.Split(excludedPkgs, ",")
	for _, ep := range eps {
		v.excludedPackages[ep] = true
	}

	return v
}

func (v *RageQuitVisitor) ProcessNextFile(f *ast.File) bool {
	if _, ok := v.excludedPackages[f.Name.Name]; ok {
		return false
	}

	v.nodeStack = nil
	v.isInMainPackage = f.Name.Name == "main"
	v.isInMainFunc = false
	return true
}

func (v *RageQuitVisitor) InspectNode(node ast.Node) bool {
	// analyze node
	if call, ok := node.(*ast.CallExpr); ok {
		v.checkRageQuitCall(call)
	}

	// manipulate stack
	if node != nil {
		v.pushNode(node)
	} else {
		v.popNode()
	}

	return true
}

func (v *RageQuitVisitor) Report(pass *analysis.Pass) {
	for _, i := range v.issues {
		pass.Report(i)
	}
}

func (v *RageQuitVisitor) pushNode(node ast.Node) {
	if _, ok := node.(*ast.File); ok {
		return
	}
	v.nodeStack = append(v.nodeStack, &node)
	if len(v.nodeStack) == 1 {
		if !v.isInMainPackage {
			return
		}
		if decl, ok := node.(*ast.FuncDecl); ok {
			if decl.Name.Name == "main" && decl.Recv == nil {
				v.isInMainFunc = true
			}
		}
	}
}

func (v *RageQuitVisitor) popNode() {
	if len(v.nodeStack) == 0 {
		return
	}
	v.nodeStack = v.nodeStack[:len(v.nodeStack)-1]
	if len(v.nodeStack) == 0 {
		v.isInMainFunc = false
	}
}

func (v *RageQuitVisitor) checkRageQuitCall(call *ast.CallExpr) {
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		v.checkRageQuitSelector(fun)
	case *ast.Ident:
		v.checkPanic(fun)
	}
}

func (v *RageQuitVisitor) checkRageQuitSelector(sel *ast.SelectorExpr) {
	if v.isInMainFunc {
		return
	}

	sname := getSelectorName(sel)
	if sname == selExit || sname == selFatal || sname == selFatalf {
		v.issues = append(v.issues,
			analysis.Diagnostic{
				Pos:     sel.Pos(),
				Message: fmt.Sprintf("%s should not be called outside main function", sname),
			})
	}
}

func (v *RageQuitVisitor) checkPanic(ident *ast.Ident) {
	if ident.Name == "panic" {
		v.issues = append(v.issues,
			analysis.Diagnostic{
				Pos:     ident.Pos(),
				Message: "panic should not be used",
			})
	}
}

func getSelectorName(sel *ast.SelectorExpr) string {
	if ident, ok := sel.X.(*ast.Ident); ok {
		return fmt.Sprintf("%s.%s", ident.Name, sel.Sel.Name)
	}
	if s, ok := sel.X.(*ast.SelectorExpr); ok {
		return fmt.Sprintf("%s.%s", getSelectorName(s), sel.Sel.Name)
	}
	if call, ok := sel.X.(*ast.CallExpr); ok {
		return fmt.Sprintf("%s.%s", getSelectorName(call.Fun.(*ast.SelectorExpr)), sel.Sel.Name)
	}
	return ""
}
