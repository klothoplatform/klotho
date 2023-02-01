package csharp

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestName(t *testing.T) {
	f, _ := NewFile("file.cs", strings.NewReader(`

global using static A.B;
global using A;
using static A.B;
using a;
using xyz = A.B;
	
namespace xyz {
	using abcdefg;
}

	`))

	i := FindImportsInFile(f)
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Print(string(b))
}
