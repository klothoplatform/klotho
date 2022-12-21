package core

import (
	"fmt"

	"go.uber.org/zap"
)

type (
	Dependency struct {
		Source ResourceKey
		Target ResourceKey
	}

	Dependencies ConcurrentMap[Dependency, struct{}]
)

func (d *Dependencies) _m() *ConcurrentMap[Dependency, struct{}] {
	return (*ConcurrentMap[Dependency, struct{}])(d)
}

func (d *Dependencies) Add(source ResourceKey, target ResourceKey) {
	m := d._m()
	m.Set(Dependency{Source: source, Target: target}, struct{}{})
	zap.S().Debugf("deps += [%s -> %s]", source, target)
}

func (d *Dependencies) Remove(source ResourceKey, target ResourceKey) bool {
	m := d._m()
	_, changed := m.Delete(Dependency{Source: source, Target: target})
	return changed
}

func (d *Dependencies) RemoveSource(source ResourceKey) (changed bool) {
	m := d._m()
	for _, dep := range m.Keys() {
		if dep.Source == source {
			m.Delete(dep)
			changed = true
		}
	}
	return
}

func (d *Dependencies) RemoveTarget(target ResourceKey) (changed bool) {
	m := d._m()
	for _, dep := range m.Keys() {
		if dep.Target == target {
			m.Delete(dep)
			changed = true
		}
	}
	return
}

func (d *Dependencies) Contains(dep Dependency) bool {
	m := d._m()
	_, contains := m.Get(dep)
	return contains
}

func (d *Dependencies) Downstream(source ResourceKey) (targets []ResourceKey) {
	m := d._m()
	for _, dep := range m.Keys() {
		if dep.Source == source {
			targets = append(targets, dep.Target)
		}
	}
	return
}

func (d *Dependencies) Upstream(target ResourceKey) (sources []ResourceKey) {
	m := d._m()
	for _, dep := range m.Keys() {
		if dep.Target == target {
			sources = append(sources, dep.Source)
		}
	}
	return
}

func (d *Dependencies) Clone() *Dependencies {
	m := d._m()
	var n ConcurrentMap[Dependency, struct{}]
	for _, dep := range m.Keys() {
		n.Set(dep, struct{}{})
	}
	return (*Dependencies)(&n)
}

func (d *Dependencies) ToArray() []Dependency {
	return d._m().Keys()
}

func (d *Dependencies) String() string {
	m := d._m()
	depsBySource := make(map[ResourceKey][]ResourceKey)
	for _, dep := range m.Keys() {
		depsBySource[dep.Source] = append(depsBySource[dep.Source], dep.Target)
	}
	s := make([]string, 0, len(depsBySource))
	for src, targets := range depsBySource {
		s = append(s, fmt.Sprintf("%s -> %v", src, targets))
	}
	return fmt.Sprintf("%v", s)
}
