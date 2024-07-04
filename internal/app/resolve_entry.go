package app

import (
	"container/heap"
	"slices"
	"time"

	"github.com/idr0id/keenctl/internal/keenetic"
)

const expirePrecision = 20 * time.Second

type resolveEntry struct {
	interfaceName string
	target        string
	resolver      string
	gateway       string
	filters       []string
	routes        []keenetic.IPRoute
	expireAt      time.Time
	auto          bool
}

func newResolveEntries(conf Config) []*resolveEntry {
	var entries []*resolveEntry
	for _, interfaceConf := range conf.Interfaces {
		entries = slices.Grow(entries, len(interfaceConf.Routes))

		for _, routeConf := range interfaceConf.Routes {
			entries = append(entries, newResolveEntry(interfaceConf, routeConf))
		}
	}
	return entries
}

func newResolveEntry(
	interfaceConf InterfaceConfig,
	routeConf RouteConfig,
) *resolveEntry {
	return &resolveEntry{
		interfaceName: interfaceConf.Name,
		gateway:       routeConf.GetGateway(interfaceConf.Defaults),
		auto:          routeConf.GetAuto(interfaceConf.Defaults),
		target:        routeConf.Target,
		resolver:      routeConf.Resolver,
		filters:       routeConf.GetFilters(interfaceConf.Defaults),
		routes:        nil,
		expireAt:      time.Unix(0, 0),
	}
}

func (e *resolveEntry) isExpired() bool {
	return time.Now().After(e.expireAt)
}

func (e *resolveEntry) applyChanges(routes []keenetic.IPRoute, expireAt time.Time) {
	e.routes = routes
	e.expireAt = expireAt.Truncate(expirePrecision).Add(expirePrecision)
}

type resolveQueue []*resolveEntry

func (q *resolveQueue) popExpiredRoutes() []*resolveEntry {
	var unresolved []*resolveEntry
	for q.Len() > 0 && (*q)[0].isExpired() {
		entry, _ := heap.Pop(q).(*resolveEntry)
		unresolved = append(unresolved, entry)
	}
	return unresolved
}

func (q *resolveQueue) Len() int {
	return len(*q)
}

func (q *resolveQueue) Less(i, j int) bool {
	left, right := (*q)[i], (*q)[j]
	return left.expireAt.Before(right.expireAt)
}

func (q *resolveQueue) Swap(i, j int) {
	(*q)[i], (*q)[j] = (*q)[j], (*q)[i]
}

func (q *resolveQueue) Push(x any) {
	entry, _ := x.(*resolveEntry)
	*q = append(*q, entry)
}

func (q *resolveQueue) Pop() any {
	old := *q
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*q = old[0 : n-1]

	return item
}
