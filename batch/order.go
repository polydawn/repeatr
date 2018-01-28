package batch

import (
	"fmt"
	"sort"

	"go.polydawn.net/go-timeless-api"
)

/*
	Compute a simple topological sort of the steps based on the wire imports.

	We break ties based on lexigraphical sort on the step names.
	We choose this simple tie-breaker rather than attempting any fancier
	logic based on e.g. downstream dependencies, etc, because ease of
	understanding and the simplicity of predicting the result of the sort
	is more important than cleverness; so is the regional stability of the
	sort in the face of changes in other parts of the graph.
*/
func orderSteps(basting api.Basting) ([]string, error) {
	result := make([]string, 0, len(basting.Steps))
	todo := make(map[string]struct{}, len(basting.Steps))
	for node := range basting.Steps {
		todo[node] = struct{}{}
	}
	edges := []string{}
	for node := range basting.Steps {
		edges = append(edges, node)
	}
	sort.Strings(edges)
	for _, node := range edges {
		if err := orderSteps_visit(node, todo, map[string]struct{}{}, &result, basting); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func orderSteps_visit(
	node string,
	todo map[string]struct{},
	loopDetector map[string]struct{},
	result *[]string,
	basting api.Basting,
) error {
	// Quick exit if possible.
	if _, ok := todo[node]; !ok {
		return nil
	}
	if _, ok := loopDetector[node]; ok {
		return fmt.Errorf("not a dag: loop detected at %q", node)
	}
	// Mark self for loop detection.
	loopDetector[node] = struct{}{}
	// Extract any imports which are dependency wiring.
	edges := []string{}
	for _, edge := range basting.Steps[node].Imports {
		if edge.CatalogName != "wire" {
			continue
		}
		depNode := string(edge.ReleaseName)
		if _, ok := basting.Steps[depNode]; !ok {
			return fmt.Errorf("invalid wire: %q has wire to non-existent %q", node, depNode)
		}
		// TODO also check output path
		edges = append(edges, depNode)
	}
	// Sort the dependency nodes by name, then recurse.
	//  This sort is necessary for deterministic order of unrelated nodes.
	sort.Strings(edges)
	for _, edge := range edges {
		if err := orderSteps_visit(edge, todo, loopDetector, result, basting); err != nil {
			return nil
		}
	}
	// Done: put this node in the results, and remove from todo set.
	*result = append(*result, node)
	delete(todo, node)
	return nil
}
