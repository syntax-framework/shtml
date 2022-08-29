package cmn

// AssetType represents the resource types handled and known by the syntax
type AssetType uint

const (
	Javascript AssetType = iota
	Stylesheet
)

// Asset a resource used by a template and mapped by syntax
type Asset struct {
	Content        []byte
	Name           string //Unique, non-conflicting name
	Size           int64
	Etag           string
	Url            string
	Type           AssetType
	Integrity      string // https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity
	CrossOrigin    string
	ReferrerPolicy string
	Filepath       string   // When the asset is a file in the file system
	Dependencies   []*Asset // Dependencies of the asset
}

func (a *Asset) Key() string {
	return a.Name
}

func (a *Asset) GetDependencies() []*Asset {
	return a.Dependencies
}

//GetId() string // d
//GetDependencies() []GNode
