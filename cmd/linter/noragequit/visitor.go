package noragequit

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

const (
	selFatal  string = "log.Fatal"
	selFatalf string = "log.Fatalf"
	selExit   string = "os.Exit"
)

type RageQuitVisitor struct {
	typesInfo *types.Info

	nodeStack       []*ast.Node
	isInMainPackage bool
	isInMainFunc    bool

	issues []analysis.Diagnostic
}

func NewRageQuitVisitor(typesInfo *types.Info) *RageQuitVisitor {
	return &RageQuitVisitor{
		typesInfo: typesInfo,
	}
}

func (v *RageQuitVisitor) ProcessNextFile(f *ast.File) {
	v.nodeStack = nil
	v.isInMainPackage = f.Name.Name == "main"
	v.isInMainFunc = false
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

	sname := v.getResolvedSelectorName(sel)
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

func (v *RageQuitVisitor) getResolvedSelectorName(sel *ast.SelectorExpr) string {
	if ident, ok := sel.X.(*ast.Ident); ok {
		qualName := ident.Name
		if pkgName, ok := v.typesInfo.Uses[ident].(*types.PkgName); ok {
			qualName = pkgName.Imported().Path()
		}
		return fmt.Sprintf("%s.%s", qualName, sel.Sel.Name)
	}
	if s, ok := sel.X.(*ast.SelectorExpr); ok {
		return fmt.Sprintf("%s.%s", v.getResolvedSelectorName(s), sel.Sel.Name)
	}
	if call, ok := sel.X.(*ast.CallExpr); ok {
		return fmt.Sprintf("%s.%s", v.getResolvedSelectorName(call.Fun.(*ast.SelectorExpr)), sel.Sel.Name)
	}
	return ""
}
