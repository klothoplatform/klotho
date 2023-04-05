package testutil

import "strings"

// UnIndent removes a level of indentation from the given string. The rules are very simple:
//   - first, drop any leading newlines from the string
//   - then, find the indentation in the first line, which is defined by the leading tabs-or-spaces in that line
//   - then, trim that prefix from all lines
//
// The prefix must match exactly, and this method removes it just by invoke [strings.CutPrefix] -- nothing fancier.
// In particular, this means that if you're mixing tabs and spaces, you may find yourself in for a bad time: if the
// first string uses "<space><tab>" and the second uses "<tab><space>", the second does not count as an indentation,
// and won't be affected.
//
// You can use this to embed yaml within test code as literal strings:
//
//	 ...
//	 SomeField: MyStruct {
//	 	foo: unIndent(`
//	 		hello: world
//	 		counts:
//	 		  - 1
//	 		  - 2
//	 		hello: world`
//	 	),
//	...
//
// The resulting string will be:
//
//	┌────────────┐ ◁─ no newline
//	│hello: world│ ◁─╮
//	│counts:     │ ◁─┤
//	│  - 1       │ ◁─┼─ no extra indentation
//	│  - 2       │ ◁─┤
//	│hello: world│ ◁─╯
//	└────────────┘
func UnIndent(y string) string {
	y = strings.TrimLeft(y, "\n")
	tabsCount := 0
	for ; tabsCount < len(y) && y[tabsCount] == '\t' || y[tabsCount] == ' '; tabsCount += 1 {
		// nothing; the tabsCount += 1 is the side effect we want
	}
	prefixTabs := y[:tabsCount]
	sb := strings.Builder{}
	sb.Grow(len(y))
	for _, line := range strings.Split(y, "\n") {
		line, _ = strings.CutPrefix(line, prefixTabs)
		sb.WriteString(line)
		sb.WriteRune('\n')
	}
	return sb.String()
}
