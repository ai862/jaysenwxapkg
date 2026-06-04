package analyze

import (
	"regexp"
	"strings"

	"github.com/dop251/goja/ast"
	"github.com/dop251/goja/parser"
)

// ExtractAPIsFromJS 通过 AST 解析 JS 代码，提取字符串字面量和对象属性中的 URL
func ExtractAPIsFromJS(content string) []string {
	program, err := parser.ParseFile(nil, "", content, 0)
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var results []string

	addURL := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		results = append(results, s)
	}

	var walk func(node interface{})
	walk = func(node interface{}) {
		if node == nil {
			return
		}

		switch n := node.(type) {

		// —— 字符字面量：提取 URL ——
		case *ast.StringLiteral:
			for _, u := range extractURLs(n.Value) {
				addURL(u)
			}

		// —— 模板字符串 ——
		case *ast.TemplateLiteral:
			for _, part := range n.Parts {
				walk(part)
			}

		// —— 二元表达式（拼接） ——
		case *ast.BinaryExpression:
			walk(n.Left)
			walk(n.Right)

		// —— 调用表达式 ——
		case *ast.CallExpression:
			walk(n.Callee)
			for _, arg := range n.ArgumentList {
				walk(arg)
			}

		// —— 对象字面量：url/api 等关键 key 的值直接提取 ——
		case *ast.ObjectLiteral:
			for _, prop := range n.Value {
				key := strings.ToLower(prop.Key)
				isURLKey := key == "url" || key == "api" || key == "baseurl" ||
					key == "base_url" || key == "uri" || key == "endpoint" ||
					key == "target" || key == "action" || key == "redirect" ||
					key == "src" || key == "href" || key == "link" ||
					key == "path" || key == "requesturl" || key == "request_url"
				if isURLKey {
					if sl, ok := prop.Value.(*ast.StringLiteral); ok {
						for _, u := range extractURLs(sl.Value) {
							addURL(u)
						}
						continue
					}
				}
				walk(prop.Value)
			}

		// —— 数组字面量 ——
		case *ast.ArrayLiteral:
			for _, v := range n.Value {
				walk(v)
			}

		// —— 函数字面量 ——
		case *ast.FunctionLiteral:
			walk(n.Body)

		// —— 函数声明 ——
		case *ast.FunctionDeclaration:
			walk(n.Function)

		// —— 表达式语句 ——
		case *ast.ExpressionStatement:
			walk(n.Expression)

		// —— 变量声明（var） ——
		case *ast.VariableStatement:
			for _, expr := range n.List {
				walk(expr)
			}
		case *ast.VariableExpression:
			walk(n.Initializer)

		// —— 块语句 ——
		case *ast.BlockStatement:
			for _, stmt := range n.List {
				walk(stmt)
			}

		// —— return ——
		case *ast.ReturnStatement:
			walk(n.Argument)

		// —— if ——
		case *ast.IfStatement:
			walk(n.Test)
			walk(n.Consequent)
			walk(n.Alternate)

		// —— 赋值 ——
		case *ast.AssignExpression:
			walk(n.Left)
			walk(n.Right)

		// —— 条件表达式 ——
		case *ast.ConditionalExpression:
			walk(n.Test)
			walk(n.Consequent)
			walk(n.Alternate)

		// —— 序列表达式 ——
		case *ast.SequenceExpression:
			for _, e := range n.Sequence {
				walk(e)
			}

		// —— 一元表达式 ——
		case *ast.UnaryExpression:
			walk(n.Operand)

		// —— new 表达式 ——
		case *ast.NewExpression:
			walk(n.Callee)
			for _, arg := range n.ArgumentList {
				walk(arg)
			}

		// —— 成员表达式 ——
		case *ast.DotExpression:
			walk(n.Left)
		case *ast.BracketExpression:
			walk(n.Left)
			walk(n.Member)

		// —— for 循环 ——
		case *ast.ForStatement:
			walk(n.Initializer)
			walk(n.Test)
			walk(n.Update)
			walk(n.Body)
		case *ast.ForInStatement:
			walk(n.Into)
			walk(n.Source)
			walk(n.Body)

		// —— while / do-while ——
		case *ast.WhileStatement:
			walk(n.Test)
			walk(n.Body)
		case *ast.DoWhileStatement:
			walk(n.Test)
			walk(n.Body)

		// —— switch ——
		case *ast.SwitchStatement:
			walk(n.Discriminant)
			for _, cs := range n.Body {
				walk(cs)
			}
		case *ast.CaseStatement:
			walk(n.Test)
			for _, stmt := range n.Consequent {
				walk(stmt)
			}

		// —— try / catch ——
		case *ast.TryStatement:
			walk(n.Body)
			if n.Catch != nil {
				walk(n.Catch)
			}
			walk(n.Finally)
		case *ast.CatchStatement:
			walk(n.Body)

		// —— throw / label / with ——
		case *ast.ThrowStatement:
			walk(n.Argument)
		case *ast.LabelledStatement:
			walk(n.Statement)
		case *ast.WithStatement:
			walk(n.Object)
			walk(n.Body)

		// —— 叶子节点 ——
		case *ast.EmptyStatement, *ast.ThisExpression, *ast.Identifier,
			*ast.NullLiteral, *ast.BooleanLiteral, *ast.NumberLiteral,
			*ast.RegExpLiteral, *ast.SuperExpression, *ast.BranchStatement:
			// 无子节点

		// —— Program（根） ——
		case *ast.Program:
			for _, stmt := range n.Body {
				walk(stmt)
			}
		}
	}

	walk(program)
	return results
}

// urlPattern 匹配 http/https/ftp/ws URL 和以 / 开头的路径
var urlPattern = regexp.MustCompile(`(?:https?|ftp|wss?)://[^\s"'<>]+|//[^\s"'<>]+|/[a-zA-Z0-9_\-./?&=%+#]+`)

// extractURLs 从字符串中提取所有 URL
func extractURLs(s string) []string {
	if len(s) < 3 {
		return nil
	}
	matches := urlPattern.FindAllString(s, -1)
	var results []string
	for _, m := range matches {
		m = strings.TrimRight(m, ".?&=/")
		if m != "" {
			results = append(results, m)
		}
	}
	return results
}
