package csharp

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestTypeDeclarations(t *testing.T) {
	f, _ := NewFile("file.cs", strings.NewReader(`

public abstract sealed class pasc1 {
}
class c1: abc.xad, def {
}

public class pc1<T> {
  class nc1 {
  
	}
	public class nc2 {
		protected class nc3 {
	    }
  	}
  }
	  
}

interface int1 {
}


public interface int2 {
}

public record r1 {
}

public struct s1 {
}
	
	
protected internal class pic1 {
}
	
internal protected class pic2 {
}
	
private protected class ppc1 {
}
	
protected private class ppc2 {
}
		
namespace ns1 {
  public record nsr1: axy, bdd {
  }
	
	namespace ns2 {
	  class ns2c {}
	}
}

	`))

	i := FindTypeDeclarationsInFile(f)
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Print(string(b))
}

func TestMethodDeclarations(t *testing.T) {
	f, _ := NewFile("file.cs", strings.NewReader(`

namespace ns1 {
namespace ns2 {
	abstract class c1 {
		public int m1() {}
		private sealed int m2() {}
		static ABC sm1(int p1, MyType p2) {}
		public abstract void am1();
		void gm1<T>(){}
	}
	
}
}

	`))

	i := FindMethodDeclarationsInFile(f)
	b, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Print(string(b))
}
