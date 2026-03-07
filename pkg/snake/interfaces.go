package snake

type Snapshottable interface {
	Snapshot() map[string]any
}
