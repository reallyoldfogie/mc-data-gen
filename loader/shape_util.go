package loader

// Vec3 is a simple 3D vector type used for world-space AABBs.
type Vec3 struct {
    X, Y, Z float64
}

// AABB represents an axis-aligned bounding box in world coordinates.
type AABB struct {
    Min Vec3
    Max Vec3
}

// IsPassable reports whether the blockstate should be considered passable
// for entity movement. This is based purely on collision, not outline.
func (info ShapeInfo) IsPassable() bool {
    if info.Air {
        return true
    }
    if !info.BlocksMovement {
        return true
    }
    return false
}

// IsStandingSurface returns the maximum Y of the collision shape in
// block-local coordinates [0,1]. Returns 0 if there is no collision.
func (info ShapeInfo) IsStandingSurface() float64 {
    top := 0.0
    for _, b := range info.Collision {
        if b.Max[1] > top {
            top = b.Max[1]
        }
    }
    return top
}

// CanSeeThrough reports whether this blockstate should be considered
// transparent for line-of-sight / vision purposes.
func (info ShapeInfo) CanSeeThrough() bool {
    if info.Air {
        return true
    }
    return !info.Opaque
}

// WorldCollisionBoxesAt returns the collision boxes for this ShapeInfo
// translated into world coordinates at the given block position.
func (info ShapeInfo) WorldCollisionBoxesAt(blockX, blockY, blockZ int) []AABB {
    if len(info.Collision) == 0 {
        return nil
    }

    bx := float64(blockX)
    by := float64(blockY)
    bz := float64(blockZ)

    out := make([]AABB, len(info.Collision))
    for i, b := range info.Collision {
        out[i] = AABB{
            Min: Vec3{X: bx + b.Min[0], Y: by + b.Min[1], Z: bz + b.Min[2]},
            Max: Vec3{X: bx + b.Max[0], Y: by + b.Max[1], Z: bz + b.Max[2]},
        }
    }
    return out
}

// IsClimbable reports whether this blockstate is climbable.
func (info ShapeInfo) IsClimbable() bool {
    return false
}

// IsClimbableFor uses block IDs to decide climbability.
func IsClimbableFor(blockID string, info ShapeInfo) bool {
    switch blockID {
    case "minecraft:ladder",
        "minecraft:vine",
        "minecraft:scaffolding",
        "minecraft:twisting_vines",
        "minecraft:weeping_vines",
        "minecraft:bubble_column":
        return true
    default:
        return false
    }
}

