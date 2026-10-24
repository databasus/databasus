// Command archlint checks the architecture rules of docs/architecture.md that golangci-lint cannot
// express. Run it from the backend module root; it prints one line per violation and fails when
// there is any.
package main

import (
	"fmt"
	"os"
)

func main() {
	violations, err := check(".", backendSpec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "archlint: %v\n", err)
		os.Exit(2)
	}

	for _, violation := range violations {
		fmt.Println(violation)
	}

	if len(violations) > 0 {
		fmt.Fprintf(os.Stderr, "archlint: %d violations\n", len(violations))
		os.Exit(1)
	}
}
