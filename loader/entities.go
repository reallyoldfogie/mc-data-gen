package loader

// EntityDimensions represents width, height, and eye height of an entity
type EntityDimensions struct {
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	EyeHeight float64 `json:"eye_height"`
	Fixed    bool    `json:"fixed"`
}

// EntityAttribute represents a single attribute with base value
type EntityAttribute struct {
	Name      string  `json:"name"`
	BaseValue float64 `json:"base_value"`
}

// EntitySizeVariant represents dimensions for a specific size value (for Slimes, etc.)
type EntitySizeVariant struct {
	Size       int               `json:"size"`
	Dimensions EntityDimensions `json:"dimensions"`
}

// EntityRecord represents a single entity type entry from entities.json
type EntityRecord struct {
	EntityID           string                      `json:"entity_id"`
	SpawnGroup         string                      `json:"spawn_group"`
	FireImmune         bool                        `json:"fire_immune"`
	DefaultDimensions  EntityDimensions            `json:"default_dimensions"`
	PoseDimensions     map[string]EntityDimensions `json:"pose_dimensions"`
	SizeVariants       []EntitySizeVariant         `json:"size_variants"`
	BabyDimensions     *EntityDimensions           `json:"baby_dimensions,omitempty"`
	Attributes         []EntityAttribute           `json:"attributes"`
	Tags               []string                    `json:"tags"`
}

// EntityRecordSlim is used in per-entity files (no EntityID)
type EntityRecordSlim struct {
	SpawnGroup         string                      `json:"spawn_group"`
	FireImmune         bool                        `json:"fire_immune"`
	DefaultDimensions  EntityDimensions            `json:"default_dimensions"`
	PoseDimensions     map[string]EntityDimensions `json:"pose_dimensions"`
	SizeVariants       []EntitySizeVariant         `json:"size_variants"`
	BabyDimensions     *EntityDimensions           `json:"baby_dimensions,omitempty"`
	Attributes         []EntityAttribute           `json:"attributes"`
	Tags               []string                    `json:"tags"`
}

// EntityFile is the per-entity file format
type EntityFile struct {
	EntityID string           `json:"entity_id"`
	Data     EntityRecordSlim `json:"data"`
}

// EntityInfo is the runtime struct returned by loader functions
type EntityInfo struct {
	ID                 string
	SpawnGroup         string
	FireImmune         bool
	DefaultDimensions  EntityDimensions
	PoseDimensions     map[string]EntityDimensions
	SizeVariants       []EntitySizeVariant
	BabyDimensions     *EntityDimensions
	Attributes         []EntityAttribute
	Tags               []string
}
