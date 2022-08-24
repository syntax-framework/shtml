package cmn

import (
	"time"
)

// AssetType represents the resource types handled and known by the syntax
type AssetType uint

const (
	Javascript AssetType = iota
	Stylesheet
)

// Asset a resource used by a template and mapped by syntax
type Asset struct {
	Content string
	Name    string
	Time    time.Time
	Size    int64
	Etag    string
	Type    AssetType
}
