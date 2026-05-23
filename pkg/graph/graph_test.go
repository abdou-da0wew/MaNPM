package graph

import (
	"testing"
)

func TestEmptyGraph(t *testing.T) {
	g := NewDependencyGraph()
	if err := g.TopologicalSort(); err != nil {
		t.Fatalf("expected no error for empty graph, got %v", err)
	}
	if len(g.Levels) != 0 {
		t.Fatalf("expected 0 levels, got %d", len(g.Levels))
	}
	if g.HasCycle() {
		t.Fatal("expected no cycle in empty graph")
	}
}

func TestSingleNode(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("a", "1.0.0", "", "", nil)
	if err := g.TopologicalSort(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(g.Levels) != 1 {
		t.Fatalf("expected 1 level, got %d", len(g.Levels))
	}
	if len(g.Levels[0]) != 1 {
		t.Fatalf("expected 1 node in level, got %d", len(g.Levels[0]))
	}
	if g.Levels[0][0].Name != "a" {
		t.Fatalf("expected node 'a', got %s", g.Levels[0][0].Name)
	}
	if g.HasCycle() {
		t.Fatal("expected no cycle for single node")
	}
}

func TestThreeLevelDAG(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("a", "1.0.0", "", "", map[string]string{})
	g.AddNode("b", "1.0.0", "", "", map[string]string{"a": "1.0.0"})
	g.AddNode("c", "1.0.0", "", "", map[string]string{"b": "1.0.0"})

	if err := g.TopologicalSort(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(g.Levels) != 3 {
		t.Fatalf("expected 3 levels, got %d", len(g.Levels))
	}

	if len(g.Levels[0]) != 1 || g.Levels[0][0].Name != "a" {
		t.Fatalf("expected level 0 to contain 'a', got %v", levelNames(g.Levels[0]))
	}
	if len(g.Levels[1]) != 1 || g.Levels[1][0].Name != "b" {
		t.Fatalf("expected level 1 to contain 'b', got %v", levelNames(g.Levels[1]))
	}
	if len(g.Levels[2]) != 1 || g.Levels[2][0].Name != "c" {
		t.Fatalf("expected level 2 to contain 'c', got %v", levelNames(g.Levels[2]))
	}

	if g.HasCycle() {
		t.Fatal("expected no cycle in DAG")
	}
}

func TestCycleDetection(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("a", "1.0.0", "", "", map[string]string{"b": "1.0.0"})
	g.AddNode("b", "1.0.0", "", "", map[string]string{"a": "1.0.0"})

	err := g.TopologicalSort()
	if err != ErrCycleDetected {
		t.Fatalf("expected ErrCycleDetected, got %v", err)
	}

	if g.Levels != nil {
		t.Fatal("expected Levels to be nil after cycle error")
	}

	if !g.HasCycle() {
		t.Fatal("expected HasCycle to return true")
	}
}

func TestSmallestLastHeuristic(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("a", "1.0.0", "", "", map[string]string{})
	g.AddNode("b", "1.0.0", "", "", map[string]string{"a": "1.0.0"})
	g.AddNode("c", "1.0.0", "", "", map[string]string{"b": "1.0.0"})

	_ = g.TopologicalSort()

	g.Nodes["node_modules/a"].Weight = 100
	g.Nodes["node_modules/b"].Weight = 50
	g.Nodes["node_modules/c"].Weight = 200

	light, heavy := g.SmallestLastHeuristic(150)

	expectedLevels := 3
	if len(light) != expectedLevels {
		t.Fatalf("expected %d lightweight levels, got %d", expectedLevels, len(light))
	}

	if len(light[0]) != 1 || light[0][0].Name != "a" {
		t.Fatalf("expected light[0] to contain 'a' (weight 100 <= 150), got %v", levelNames(light[0]))
	}
	if len(light[1]) != 1 || light[1][0].Name != "b" {
		t.Fatalf("expected light[1] to contain 'b' (weight 50 <= 150), got %v", levelNames(light[1]))
	}
	if len(light[2]) != 0 {
		t.Fatalf("expected light[2] to be empty (c weight 200 > 150), got %v", levelNames(light[2]))
	}

	if len(heavy) != 1 || heavy[0].Name != "c" {
		t.Fatalf("expected heavyweight to contain 'c', got %v", nodeNames(heavy))
	}
}

func TestSmallestLastHeuristicSortOrder(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("a", "1.0.0", "", "", nil)
	g.AddNode("b", "1.0.0", "", "", map[string]string{"a": "1.0.0"})
	g.AddNode("c", "1.0.0", "", "", map[string]string{"a": "1.0.0"})
	g.AddNode("d", "1.0.0", "", "", map[string]string{"b": "1.0.0", "c": "1.0.0"})

	_ = g.TopologicalSort()

	g.Nodes["node_modules/a"].Weight = 300
	g.Nodes["node_modules/b"].Weight = 50
	g.Nodes["node_modules/c"].Weight = 100
	g.Nodes["node_modules/d"].Weight = 200

	light, heavy := g.SmallestLastHeuristic(250)

	if len(light) != 3 {
		t.Fatalf("expected 3 levels, got %d", len(light))
	}

	if len(light[0]) != 0 {
		t.Fatalf("expected light[0] to be empty (a weight 300 > 250), got %v", levelNames(light[0]))
	}

	if len(light[1]) != 2 {
		t.Fatalf("expected 2 nodes in light[1], got %d", len(light[1]))
	}
	if light[1][0].Name != "b" || light[1][1].Name != "c" {
		t.Fatalf("expected light[1] sorted by weight: b(50), c(100), got %v", levelNames(light[1]))
	}

	if len(light[2]) != 1 || light[2][0].Name != "d" {
		t.Fatalf("expected light[2] to contain 'd', got %v", levelNames(light[2]))
	}

	if len(heavy) != 1 || heavy[0].Name != "a" {
		t.Fatalf("expected heavyweight to contain 'a', got %v", nodeNames(heavy))
	}
}

func TestSelfDependency(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("a", "1.0.0", "", "", map[string]string{"a": "1.0.0"})

	err := g.TopologicalSort()
	if err != ErrCycleDetected {
		t.Fatalf("expected ErrCycleDetected for self-dependency, got %v", err)
	}
}

func TestMultipleRoots(t *testing.T) {
	g := NewDependencyGraph()
	g.AddNode("a", "1.0.0", "", "", nil)
	g.AddNode("b", "1.0.0", "", "", nil)
	g.AddNode("c", "1.0.0", "", "", map[string]string{"a": "1.0.0", "b": "1.0.0"})

	if err := g.TopologicalSort(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(g.Levels) != 2 {
		t.Fatalf("expected 2 levels, got %d", len(g.Levels))
	}

	rootNames := levelNames(g.Levels[0])
	if len(rootNames) != 2 {
		t.Fatalf("expected 2 root nodes, got %d", len(rootNames))
	}

	if len(g.Levels[1]) != 1 || g.Levels[1][0].Name != "c" {
		t.Fatalf("expected level 1 to contain 'c', got %v", levelNames(g.Levels[1]))
	}
}

func levelNames(nodes []*PackageNode) []string {
	names := make([]string, len(nodes))
	for i, n := range nodes {
		names[i] = n.Name
	}
	return names
}

func nodeNames(nodes []*PackageNode) []string {
	names := make([]string, len(nodes))
	for i, n := range nodes {
		names[i] = n.Name
	}
	return names
}
