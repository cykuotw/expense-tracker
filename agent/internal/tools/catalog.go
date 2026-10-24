package tools

import (
	"fmt"
	"maps"
	"slices"

	"google.golang.org/adk/v2/tool"
)

// Adding a backend capability requires an explicit implementation in this catalog.
var catalog = map[string]func(BalanceClient) (tool.Tool, error){
	"get_balance": NewBalance,
}

// Names lists the tools available for validated skill bindings.
func Names() []string {
	return slices.Sorted(maps.Keys(catalog))
}

// Build creates a registered tool with its injected backend dependency.
func Build(name string, client BalanceClient) (tool.Tool, error) {
	factory, known := catalog[name]
	if !known {
		return nil, fmt.Errorf("unknown tool %q", name)
	}
	return factory(client)
}
