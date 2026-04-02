package a2uiruntime_test

import (
	"fmt"

	"github.com/tmc/swiftui/a2uiruntime"
)

func Example() {
	rt := a2uiruntime.New()
	matrix := rt.SupportMatrix()
	fmt.Println(matrix.Catalogs[0])
	fmt.Println(matrix.Extensions[0] != "")
	// Output:
	// https://a2ui.org/specification/v0_9/basic_catalog.json
	// true
}
