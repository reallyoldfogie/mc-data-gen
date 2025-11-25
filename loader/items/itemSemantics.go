package items

import (
	"strings"
)

// ----- Raw JSON types -----

type FoodComponent struct {
	Nutrition    int     `json:"nutrition"`
	Saturation   float64 `json:"saturation"`
	CanAlwaysEat bool    `json:"can_always_eat"`
}

type Components struct {
	MaxDamage      *int           `json:"max_damage,omitempty"`
	MaxDamageStack *int           `json:"max_damage_stack,omitempty"`
	Damage         *int           `json:"damage,omitempty"`
	Enchantments   []any          `json:"enchantments,omitempty"` // you can refine this later
	IsTool         *bool          `json:"is_tool,omitempty"`
	Food           *FoodComponent `json:"food,omitempty"`
	// Extend as you add more fields to components
}

// This reflects a single entry like minecraft:iron_sword
type ItemRecord struct {
	ID             string     `json:"id"`
	MaxStackSize   int        `json:"max_stack_size"`
	TranslationKey string     `json:"translation_key"`
	Rarity         string     `json:"rarity"`
	UseAnimation   string     `json:"use_animation"`
	Tags           []string   `json:"tags"`
	Components     Components `json:"components"`

	// You already have these in the JSON; we can use them as hints
	IsWeapon bool `json:"is_weapon"`
	IsFood   bool `json:"is_food"`
}

// ItemSemantics is the unified, tag/component-driven meaning of an item.
type ItemSemantics struct {
	// Core combat/tools
	IsTool         bool
	IsWeapon       bool
	IsMeleeWeapon  bool
	IsRangedWeapon bool
	IsArmor        bool
	IsShield       bool

	// Consumables / usage
	IsFood       bool
	IsPotion     bool
	IsThrowable  bool // snowball, egg, ender pearl, splash potion, etc.
	IsProjectile bool // arrows, spectral arrows
	IsUsable     bool // has a meaningful use action/duration

	// World interaction
	IsBlockItem bool // places a block (stone, wood, etc.)

	// Resource / crafting
	IsIngredient     bool // ores, ingots, dusts, crops, stones, etc.
	IsFoodIngredient bool // wheat, sugar, etc.
	IsCropProduct    bool // wheat, carrots, potatoes

	// Misc special categories
	IsMusicDisc     bool
	IsBannerPattern bool
	IsSpawnEgg      bool
	IsBoat          bool
	IsMinecart      bool

	// Debug / introspection
	SourceTags       []string
	SourceComponents Components
}

type Tag struct {
	Namespace string
	Path      string // "tools/melee_weapon"
}

func ParseTag(raw string) Tag {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return Tag{Namespace: "", Path: raw}
	}
	return Tag{
		Namespace: parts[0],
		Path:      strings.TrimPrefix(parts[1], "/"),
	}
}

func buildTagSet(raw []string) TagSet {
	ts := TagSet{
		All:       make([]Tag, 0, len(raw)),
		ByNS:      map[string][]Tag{},
		ByNSIndex: map[string]map[string]Tag{},
	}
	for _, r := range raw {
		t := ParseTag(r)
		ts.All = append(ts.All, t)
		ts.ByNS[t.Namespace] = append(ts.ByNS[t.Namespace], t)
		if ts.ByNSIndex[t.Namespace] == nil {
			ts.ByNSIndex[t.Namespace] = map[string]Tag{}
		}
		ts.ByNSIndex[t.Namespace][t.Path] = t
	}
	return ts
}

// You call this once per ItemInfo after loading JSON.
// DeriveSemantics computes the semantics from one ItemRecord.
func DeriveSemantics(rec ItemRecord) ItemSemantics {
	tags := buildTagSet(rec.Tags)

	s := ItemSemantics{
		SourceTags:       append([]string(nil), rec.Tags...),
		SourceComponents: rec.Components,
	}

	// Seed with what the exporter already knows
	if rec.IsWeapon {
		s.IsWeapon = true
	}
	if rec.IsFood {
		s.IsFood = true
	}
	if rec.Components.IsTool != nil && *rec.Components.IsTool {
		s.IsTool = true
	}

	applyTagRules(&s, rec, tags)
	applyComponentRules(&s, rec)
	postProcess(&s)

	return s
}

func applyTagRules(s *ItemSemantics, rec ItemRecord, tags TagSet) {
	// ----- Tools / Weapons -----

	// Common tags: c:tools, c:tools/*
	if tags.HasPath("c", "tools") || tags.HasPathPrefix("c", "tools/") {
		s.IsTool = true
	}

	// Melee weapons: c:tools/melee_weapon or c:tools/melee_weapons
	if tags.HasPath("c", "tools/melee_weapon") || tags.HasPath("c", "tools/melee_weapons") {
		s.IsWeapon = true
		s.IsMeleeWeapon = true
	}

	// Vanilla swords tag
	if tags.HasPath("minecraft", "swords") {
		s.IsWeapon = true
		s.IsMeleeWeapon = true
	}

	// Enchantable weapon-ish tags
	if tags.HasPath("minecraft", "enchantable/weapon") ||
		tags.HasPath("minecraft", "enchantable/sword") ||
		tags.HasPath("minecraft", "enchantable/sharp_weapon") {
		s.IsWeapon = true
	}

	// You could add more here: bows, crossbows, tridents, etc.

	// ----- Food / food-ish -----

	// Any food-ish tags
	if tags.HasPathPrefix("minecraft", "food") ||
		tags.HasPathPrefix("minecraft", "fishes") ||
		tags.HasPath("minecraft", "cat_food") ||
		tags.HasPath("minecraft", "ocelot_food") ||
		tags.HasPathPrefix("c", "foods") {
		s.IsFood = true
	}

	// Distinguish consumable food vs ingredients:
	if tags.HasPathPrefix("c", "foods/") {
		s.IsFood = true
	}
	// “animal food” but maybe not edible by player (you can adjust)
	if tags.HasPath("c", "animal_foods") {
		s.IsFoodIngredient = true
	}

	// ----- Block items / ingredients -----

	// Stones almost certainly are block items + ingredients
	if tags.HasPath("c", "stones") || tags.HasPathPrefix("c", "ore_bearing_ground/") {
		s.IsBlockItem = true
		s.IsIngredient = true
	}

	// Heuristic: if translation key starts with block.minecraft.*, probably a block item
	if strings.HasPrefix(rec.TranslationKey, "block.") {
		s.IsBlockItem = true
	}

	// Generic “ingredient” heuristics from tags:
	if tags.HasPathPrefix("c", "ingots") ||
		tags.HasPathPrefix("c", "ores") ||
		tags.HasPathPrefix("c", "dusts") {
		s.IsIngredient = true
	}

	// ----- Projectiles / throwables etc. -----
	// (You don’t have examples yet, but here’s where you’d wire:
	//   c:arrows, c:throwables, minecraft:arrows, etc.)
}

func applyComponentRules(s *ItemSemantics, rec ItemRecord) {
	// Food component → definitely edible
	if rec.Components.Food != nil {
		s.IsFood = true
		// For some use-cases, food items are also ingredients
		// s.IsIngredient = true
	}

	// is_tool component → definitely a tool
	if rec.Components.IsTool != nil && *rec.Components.IsTool {
		s.IsTool = true
	}

	// You can check max_damage > 0 to bias toward tools/weapons:
	if rec.Components.MaxDamage != nil && *rec.Components.MaxDamage > 0 {
		// Don't auto-flag as weapon, but we know it's a durable item.
		// Could choose to treat “durable with weapon-like tags” as weapon++
	}

	// Use animation → some sort of usable item
	switch rec.UseAnimation {
	case "EAT", "DRINK", "BLOCK", "BOW", "SPEAR":
		s.IsUsable = true
	}
}

func postProcess(s *ItemSemantics) {
	// If it’s explicitly melee or ranged, mark as a weapon
	if s.IsMeleeWeapon || s.IsRangedWeapon {
		s.IsWeapon = true
	}

	// Example heuristic: projectiles are usually throwable or fired by tools
	if s.IsProjectile {
		// you could bias this toward ranged behavior
	}

	// If it’s a block item but we never called it an ingredient,
	// we can reasonably treat building blocks as ingredients too.
	if s.IsBlockItem && !s.IsIngredient {
		s.IsIngredient = true
	}
}

// Small struct for convenience: group tags by namespace.
type TagSet struct {
	All       []Tag
	ByNS      map[string][]Tag
	ByNSIndex map[string]map[string]Tag // ns -> path -> Tag
}

func (ts TagSet) HasPath(ns, path string) bool {
	if nsMap, ok := ts.ByNSIndex[ns]; ok {
		_, ok := nsMap[path]
		return ok
	}
	return false
}

func (ts TagSet) HasPathPrefix(ns, prefix string) bool {
	for _, t := range ts.ByNS[ns] {
		if strings.HasPrefix(t.Path, prefix) {
			return true
		}
	}
	return false
}

// Any tag containing substring (for crude heuristics)
func (ts TagSet) HasSubstr(substr string) bool {
	for _, t := range ts.All {
		if strings.Contains(t.Namespace+":"+t.Path, substr) {
			return true
		}
	}
	return false
}
