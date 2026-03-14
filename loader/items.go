package loader

// FoodComponent represents food data from items.json
type FoodComponent struct {
	Nutrition    int     `json:"nutrition"`
	Saturation   float64 `json:"saturation"`
	CanAlwaysEat bool    `json:"can_always_eat"`
}

// ItemComponents represents all components of an item
type ItemComponents struct {
	MaxDamage      *int            `json:"max_damage,omitempty"`
	MaxDamageStack *int            `json:"max_damage_stack,omitempty"`
	Damage         *int            `json:"damage,omitempty"`
	Enchantments   []any           `json:"enchantments,omitempty"`
	IsTool         *bool           `json:"is_tool,omitempty"`
	Food           *FoodComponent  `json:"food,omitempty"`
}

// ItemRecord represents a single item entry from items.json
type ItemRecord struct {
	ID             string         `json:"id"`
	MaxStackSize   int            `json:"max_stack_size"`
	TranslationKey string         `json:"translation_key"`
	Rarity         string         `json:"rarity"`
	Fireproof      bool           `json:"fireproof"`
	UseAnimation   string         `json:"use_animation"`
	Tags           []string       `json:"tags"`
	Components     ItemComponents `json:"components"`
	IsWeapon       bool           `json:"is_weapon"`
	IsFood         bool           `json:"is_food"`
}

// ItemRecordSlim is used in per-item files (no ID)
type ItemRecordSlim struct {
	MaxStackSize   int            `json:"max_stack_size"`
	TranslationKey string         `json:"translation_key"`
	Rarity         string         `json:"rarity"`
	Fireproof      bool           `json:"fireproof"`
	UseAnimation   string         `json:"use_animation"`
	Tags           []string       `json:"tags"`
	Components     ItemComponents `json:"components"`
	IsWeapon       bool           `json:"is_weapon"`
	IsFood         bool           `json:"is_food"`
}

// ItemFile is the per-item file format
type ItemFile struct {
	ItemID string        `json:"item_id"`
	Data   ItemRecordSlim `json:"data"`
}

// ItemInfo is the runtime struct returned by loader functions
type ItemInfo struct {
	ID             string
	MaxStackSize   int
	TranslationKey string
	Rarity         string
	Fireproof      bool
	UseAnimation   string
	Tags           []string
	Components     ItemComponents
	IsWeapon       bool
	IsFood         bool
}
