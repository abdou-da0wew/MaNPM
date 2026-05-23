package graph

import (
	"errors"
	"path/filepath"
	"sort"
	"sync"
)

var ErrCycleDetected = errors.New("dependency cycle detected")

type PackageNode struct {
	Name         string
	Version      string
	Resolved     string
	Integrity    string
	Dependencies map[string]string
	Weight       int64
}

type DependencyGraph struct {
	mu     sync.RWMutex
	Nodes  map[string]*PackageNode
	Levels [][]*PackageNode
}

func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		Nodes: make(map[string]*PackageNode),
	}
}

func (g *DependencyGraph) AddNode(name, version, resolved, integrity string, deps map[string]string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	key := filepath.Join("node_modules", name)
	g.Nodes[key] = &PackageNode{
		Name:         name,
		Version:      version,
		Resolved:     resolved,
		Integrity:    integrity,
		Dependencies: deps,
	}
}

func (g *DependencyGraph) TopologicalSort() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Levels = nil

	dependents := make(map[string][]string)
	inDegree := make(map[string]int)

	for key := range g.Nodes {
		inDegree[key] = 0
	}

	for key, node := range g.Nodes {
		for depName := range node.Dependencies {
			depKey := filepath.Join("node_modules", depName)
			if _, exists := g.Nodes[depKey]; exists {
				inDegree[key]++
				dependents[depKey] = append(dependents[depKey], key)
			}
		}
	}

	queue := make([]string, 0)
	for key := range g.Nodes {
		if inDegree[key] == 0 {
			queue = append(queue, key)
		}
	}

	visited := 0
	for len(queue) > 0 {
		level := make([]*PackageNode, len(queue))
		for i, key := range queue {
			level[i] = g.Nodes[key]
		}
		g.Levels = append(g.Levels, level)

		nextQueue := make([]string, 0)
		for _, key := range queue {
			visited++
			for _, dependent := range dependents[key] {
				inDegree[dependent]--
				if inDegree[dependent] == 0 {
					nextQueue = append(nextQueue, dependent)
				}
			}
		}
		queue = nextQueue
	}

	if visited != len(g.Nodes) {
		g.Levels = nil
		return ErrCycleDetected
	}
	return nil
}

func (g *DependencyGraph) SmallestLastHeuristic(threshold int64) (lightweight [][]*PackageNode, heavyweight []*PackageNode) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	lightweight = make([][]*PackageNode, len(g.Levels))
	for i, level := range g.Levels {
		sorted := make([]*PackageNode, len(level))
		copy(sorted, level)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Weight < sorted[j].Weight
		})

		lightLevel := make([]*PackageNode, 0, len(sorted))
		for _, node := range sorted {
			if node.Weight <= threshold {
				lightLevel = append(lightLevel, node)
			} else {
				heavyweight = append(heavyweight, node)
			}
		}
		lightweight[i] = lightLevel
	}
	return
}

func (g *DependencyGraph) HasCycle() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	const (
		white = 0
		gray  = 1
		black = 2
	)

	color := make(map[string]int)
	for key := range g.Nodes {
		color[key] = white
	}

	var dfs func(key string) bool
	dfs = func(key string) bool {
		color[key] = gray
		node := g.Nodes[key]
		for depName := range node.Dependencies {
			depKey := filepath.Join("node_modules", depName)
			if _, exists := g.Nodes[depKey]; !exists {
				continue
			}
			if color[depKey] == gray {
				return true
			}
			if color[depKey] == white {
				if dfs(depKey) {
					return true
				}
			}
		}
		color[key] = black
		return false
	}

	for key := range g.Nodes {
		if color[key] == white {
			if dfs(key) {
				return true
			}
		}
	}
	return false
}
