package com.example.dataexporter;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonArray;
import com.google.gson.JsonObject;
import net.fabricmc.api.ModInitializer;
import net.fabricmc.fabric.api.event.lifecycle.v1.ServerLifecycleEvents;

import net.minecraft.block.Block;
import net.minecraft.block.BlockState;
import net.minecraft.block.ShapeContext;
import net.minecraft.component.ComponentType;
import net.minecraft.component.DataComponentTypes;
import net.minecraft.entity.Entity;
import net.minecraft.entity.EntityPose;
import net.minecraft.entity.EntityType;
import net.minecraft.entity.LivingEntity;
import net.minecraft.entity.SpawnReason;
import net.minecraft.entity.attribute.DefaultAttributeContainer;
import net.minecraft.entity.attribute.EntityAttribute;
import com.mojang.authlib.GameProfile;
import net.minecraft.server.network.ServerPlayerEntity;
import net.minecraft.entity.attribute.EntityAttributeInstance;
import net.minecraft.entity.attribute.EntityAttributes;
import net.minecraft.entity.mob.MobEntity;
import net.minecraft.entity.mob.SlimeEntity;
import net.minecraft.entity.passive.PassiveEntity;
import net.minecraft.entity.passive.PufferfishEntity;
import net.minecraft.entity.mob.PhantomEntity;
import net.minecraft.item.Item;
import net.minecraft.item.ItemStack;
import net.minecraft.registry.Registries;
import net.minecraft.registry.entry.RegistryEntry;
import net.minecraft.registry.tag.BlockTags;
import net.minecraft.registry.tag.FluidTags;
import net.minecraft.server.MinecraftServer;
import net.minecraft.server.world.ServerWorld;
import net.minecraft.state.property.Property;
import net.minecraft.util.Identifier;
import net.minecraft.util.Rarity;
import net.minecraft.util.math.BlockPos;
import net.minecraft.util.math.Box;
import net.minecraft.util.shape.VoxelShape;
import net.minecraft.util.Unit;
import net.minecraft.world.World;

import java.io.IOException;
import java.io.OutputStreamWriter;
import java.lang.reflect.Field;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.UUID;
import java.util.Collections;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.TreeMap;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class DataExporterMod implements ModInitializer {

    private static final Logger LOGGER = LoggerFactory.getLogger("data_exporter");

    private static final Gson GSON = new GsonBuilder()
            .setPrettyPrinting()
            .disableHtmlEscaping()
            .create();

    @Override
    public void onInitialize() {
        LOGGER.info("[DataExporter] onInitialize");

        // Run once when the dedicated server has fully started.
        ServerLifecycleEvents.SERVER_STARTED.register(server -> {
            LOGGER.info("[DataExporter] SERVER_STARTED callback");
            try {
                dumpAllBlockStates(server);
                dumpItems(server);
                dumpEntities(server);
                dumpPoses(server);
            } catch (IOException e) {
                LOGGER.error("[DataExporter] Failed to dump data", e);
            } catch (IllegalAccessException e) {
                LOGGER.error("[DataExporter] Failed to dump data", e);
            } finally {
                LOGGER.info("[DataExporter] Stopping server after export");
                // In 1.21.x, stop(boolean waitForShutdown)
                server.stop(false);
            }
        });
    }

    private void dumpAllBlockStates(MinecraftServer server) throws IOException {
        Path runDir = server.getRunDirectory(); // This is a Path in 1.21.1+
        Path outDir = runDir.resolve("data");
        Files.createDirectories(outDir);
        Path outFile = outDir.resolve("blocks.json");

        LOGGER.info("[DataExporter] Writing block data to {}", outFile.toAbsolutePath());

        // Collect all block entries first, then sort by block ID
        List<JsonObject> allEntries = new ArrayList<>();

        var world = server.getOverworld();
        BlockPos pos = BlockPos.ORIGIN;
        ShapeContext ctx = ShapeContext.absent();

        for (Block block : Registries.BLOCK) {
            Identifier id = Registries.BLOCK.getId(block);

            for (BlockState state : block.getStateManager().getStates()) {
                JsonObject entry = new JsonObject();
                entry.addProperty("block_id", id.toString());
                entry.add("properties", serializeProperties(state));

                VoxelShape collision = state.getCollisionShape(world, pos, ctx);
                entry.add("collision_boxes", serializeShape(collision));

                VoxelShape outline = state.getOutlineShape(world, pos, ctx);
                entry.add("outline_boxes", serializeShape(outline));

                entry.addProperty("air", state.isAir());
                entry.addProperty("opaque", state.isOpaque());
                entry.addProperty("solid_block", state.isSolidBlock(world, pos));
                entry.addProperty("replaceable", state.isReplaceable());
                entry.addProperty("blocks_movement", !collision.isEmpty());

                // --- TAG-BASED SEMANTICS ---

                // climbable blocks (#minecraft:climbable)
                boolean climbable = state.isIn(BlockTags.CLIMBABLE);
                entry.addProperty("climbable", climbable);

                // door-like blocks (doors, trapdoors, fence gates)
                boolean doorLike = state.isIn(BlockTags.DOORS)
                        || state.isIn(BlockTags.TRAPDOORS)
                        || state.isIn(BlockTags.FENCE_GATES);
                entry.addProperty("door_like", doorLike);

                // fence-like blocks (fences, walls)
                boolean fenceLike = state.isIn(BlockTags.FENCES)
                        || state.isIn(BlockTags.WALLS);
                entry.addProperty("fence_like", fenceLike);

                // slabs / stairs
                boolean slab = state.isIn(BlockTags.SLABS);
                boolean stair = state.isIn(BlockTags.STAIRS);
                entry.addProperty("slab", slab);
                entry.addProperty("stair", stair);

                // logs / leaves (for tree / foliage detection)
                boolean logOrLeaf = state.isIn(BlockTags.LOGS)
                        || state.isIn(BlockTags.LEAVES);
                entry.addProperty("log_or_leaf", logOrLeaf);

                // Fluids: check the fluid state
                var fluidState = state.getFluidState();
                boolean isWater = fluidState.isIn(FluidTags.WATER);
                boolean isLava = fluidState.isIn(FluidTags.LAVA);
                boolean isFluid = !fluidState.isEmpty();

                entry.addProperty("water", isWater);
                entry.addProperty("lava", isLava);
                entry.addProperty("fluid", isFluid);

                // --- END TAG SEMANTICS ---

                // --- BLOCK-LEVEL PROPERTIES (same for all states of a block) ---
                float hardness = state.getHardness(world, pos);
                entry.addProperty("hardness", hardness);
                entry.addProperty("resistance", block.getBlastResistance());

                Item blockItem = block.asItem();
                int stackSize = (blockItem != null && blockItem != net.minecraft.item.Items.AIR)
                        ? blockItem.getMaxCount() : 0;
                entry.addProperty("stack_size", stackSize);

                entry.addProperty("diggable", hardness >= 0);

                JsonArray materialArray = new JsonArray();
                if (state.isIn(BlockTags.PICKAXE_MINEABLE)) materialArray.add("mineable/pickaxe");
                if (state.isIn(BlockTags.AXE_MINEABLE)) materialArray.add("mineable/axe");
                if (state.isIn(BlockTags.SHOVEL_MINEABLE)) materialArray.add("mineable/shovel");
                if (state.isIn(BlockTags.HOE_MINEABLE)) materialArray.add("mineable/hoe");
                entry.add("material", materialArray);
                // --- END BLOCK-LEVEL PROPERTIES ---

                allEntries.add(entry);
            }
        }

        // Sort by block_id for deterministic output
        allEntries.sort(Comparator.comparing(e -> e.get("block_id").getAsString()));

        // Convert to JsonArray
        JsonArray allStates = new JsonArray();
        for (JsonObject entry : allEntries) {
            allStates.add(entry);
        }

        try (var writer = new OutputStreamWriter(
                Files.newOutputStream(outFile),
                StandardCharsets.UTF_8)) {
            GSON.toJson(allStates, writer);
        }

        LOGGER.info("[DataExporter] Finished writing {} blockstates", allStates.size());
    }

    private void dumpItems(MinecraftServer server) throws IOException, IllegalAccessException {
        Path runDir = server.getRunDirectory(); // This is a Path in 1.21.1+
        Path outDir = runDir.resolve("data");
        Files.createDirectories(outDir);
        Path outFile = outDir.resolve("items.json");
        List<Map<String, Object>> allItems = new ArrayList<>();

        LOGGER.info("[DataExporter] Starting item export");
        for (Identifier id : Registries.ITEM.getIds()) {
            Item item = Registries.ITEM.get(id);
            Map<String, Object> info = collectItemInfo(id, item, server);
            allItems.add(info);
        }

        allItems.sort(Comparator.comparing(item -> item.get("id").toString()));
        try (var writer = new OutputStreamWriter(
                Files.newOutputStream(outFile),
                StandardCharsets.UTF_8)) {
            GSON.toJson(allItems, writer);
        }
        LOGGER.info("[DataExporter] Finished writing {} items", allItems.size());
    }

    private void dumpPoses(MinecraftServer server) throws IOException {
        Path runDir = server.getRunDirectory();
        Path outDir = runDir.resolve("data");
        Files.createDirectories(outDir);
        Path outFile = outDir.resolve("poses.json");

        // EntityPose.ordinal() is the wire varint sent in the Pose metadata
        // field (handler type id 21). We dump an explicit ordinal->name map
        // rather than a positional array so the on-disk format survives any
        // future reordering of these keys (the loader trusts the key, not the
        // file position).
        LinkedHashMap<String, String> poseMap = new LinkedHashMap<>();
        for (EntityPose pose : EntityPose.values()) {
            poseMap.put(Integer.toString(pose.ordinal()), pose.name().toLowerCase());
        }

        LOGGER.info("[DataExporter] Writing {} entity poses to {}", poseMap.size(), outFile.toAbsolutePath());
        try (var writer = new OutputStreamWriter(
                Files.newOutputStream(outFile),
                StandardCharsets.UTF_8)) {
            GSON.toJson(poseMap, writer);
        }
    }

    private void dumpEntities(MinecraftServer server) throws IOException {
        Path runDir = server.getRunDirectory();
        Path outDir = runDir.resolve("data");
        Files.createDirectories(outDir);
        Path outFile = outDir.resolve("entities.json");

        ServerWorld world = server.getOverworld();
        List<Map<String, Object>> allEntities = new ArrayList<>();

        LOGGER.info("[DataExporter] Starting entity export");
        for (EntityType<?> entityType : Registries.ENTITY_TYPE) {
            Identifier id = Registries.ENTITY_TYPE.getId(entityType);
            Map<String, Object> info = new LinkedHashMap<>();
            info.put("entity_id", id.toString());
            info.put("spawn_group", entityType.getSpawnGroup().name());
            info.put("fire_immune", entityType.isFireImmune());

            var defaultDims = entityType.getDimensions();
            info.put("default_dimensions", serializeEntityDimensions(defaultDims));

            String entityId = id.toString();
            Entity entity = null;

            // Special case: player entity
            if (entityId.equals("minecraft:player")) {
                try {
                    GameProfile dummyProfile = new GameProfile(UUID.randomUUID(), "DataGenPlayer");
                    // ServerPlayerEntity's constructor eagerly reads clientOptions.language()
                    // during construction, so passing null (as this used to) always throws a
                    // NullPointerException here — use the client-facing default instead.
                    entity = new ServerPlayerEntity(world.getServer(), world, dummyProfile,
                            net.minecraft.network.packet.c2s.common.SyncedClientOptions.createDefault());
                    LOGGER.info("[DataExporter] Entity instantiation OK: {}", id);
                } catch (Exception e) {
                    LOGGER.info("[DataExporter] Entity instantiation FAILED: {} - {}", id, e.getMessage());
                    entity = null;
                }
            } else {
                entity = EntityLookupFactory.createEntity(entityType, world);
                if (entity == null) {
                    LOGGER.info("[DataExporter] Entity instantiation FAILED: {}", id);
                } else {
                    LOGGER.info("[DataExporter] Entity instantiation OK: {}", id);
                }
            }

            // Extract pose dimensions - only include non-default poses
            Map<String, Object> poseDims = new TreeMap<>();
            if (entity != null) {
                for (EntityPose pose : EntityPose.values()) {
                    try {
                        var dims = entity.getDimensions(pose);
                        String poseName = pose.name().toLowerCase();

                        // Skip the hardcoded fallback (0.2 x 0.2 x 0.2, fixed=true) that Minecraft
                        // returns for unsupported poses
                        if (dims.width() == 0.2f && dims.height() == 0.2f && dims.eyeHeight() == 0.2f && dims.fixed()) {
                            continue;
                        }

                        // Only include if different from default
                        if (dims.width() != defaultDims.width() || dims.height() != defaultDims.height()
                                || dims.eyeHeight() != defaultDims.eyeHeight()) {
                            poseDims.put(poseName, serializeEntityDimensions(dims));
                        }
                    } catch (Exception e) {
                        // Skip poses that don't exist for this entity
                    }
                }
            } else if (entityId.equals("minecraft:player")) {
                // Player fallback: add known player poses since player couldn't be instantiated
                poseDims.put("standing", serializeEntityDimensions(defaultDims));
                // Player can also crouch
                Map<String, Object> crouchDims = new LinkedHashMap<>();
                crouchDims.put("width", 0.6);
                crouchDims.put("height", 1.5);
                crouchDims.put("eye_height", 1.27);
                crouchDims.put("fixed", false);
                poseDims.put("crouching", crouchDims);
                // Swimming
                Map<String, Object> swimDims = new LinkedHashMap<>();
                swimDims.put("width", 0.6);
                swimDims.put("height", 0.6);
                swimDims.put("eye_height", 0.6);
                swimDims.put("fixed", false);
                poseDims.put("swimming", swimDims);
            }
            info.put("pose_dimensions", poseDims);

            // Try to capture attributes and size variants by instantiating
            try {
                if (entity != null) {
                    // Only try to extract attributes if entity is a LivingEntity
                    List<Map<String, Object>> attributes = Collections.emptyList();
                    if (entity instanceof LivingEntity livingEntity) {
                        attributes = extractAttributes(livingEntity);
                    }
                    info.put("attributes", attributes);

                    // Get size variants for special cases
                    List<Map<String, Object>> sizeVariants = extractSizeVariants(entity);
                    info.put("size_variants", sizeVariants);

                    // Get baby dimensions if applicable
                    if (entity instanceof PassiveEntity animal) {
                        animal.setBaby(true);
                        var babyDimentions = animal.getDimensions(EntityPose.STANDING);
                        if (babyDimentions != defaultDims) {
                            info.put("baby_dimensions",
                                    serializeEntityDimensions(babyDimentions));
                        }
                    }
                } else {
                    info.put("attributes", Collections.emptyList());
                    info.put("size_variants", Collections.emptyList());
                }
            } catch (Exception e) {
                LOGGER.info("[DataExporter] Could not instantiate {}: {}", id, e.getMessage());
                info.put("attributes", Collections.emptyList());
                info.put("size_variants", Collections.emptyList());
            } finally {
                if (entity != null) {
                    entity.discard();
                }
            }

            List<String> tags = new ArrayList<>();
            RegistryEntry<EntityType<?>> entry = Registries.ENTITY_TYPE.getEntry(entityType);
            entry.streamTags().forEach(tagKey -> tags.add(tagKey.id().toString()));
            Collections.sort(tags);
            info.put("tags", tags);

            allEntities.add(info);
        }

        allEntities.sort(Comparator.comparing(e -> e.get("entity_id").toString()));
        try (var writer = new OutputStreamWriter(Files.newOutputStream(outFile), StandardCharsets.UTF_8)) {
            GSON.toJson(allEntities, writer);
        }
        LOGGER.info("[DataExporter] Finished writing {} entities", allEntities.size());
    }

    // Attribute registry ids dropped the "generic." namespace prefix in Minecraft
    // 1.21.2 (e.g. "minecraft:generic.max_health" -> "minecraft:max_health"). This
    // project targets both eras (1.21.1 is pre-rename; 1.21.2+ is post-rename), so
    // each attribute is looked up by its modern id first, falling back to the
    // legacy id if the modern one isn't registered. The output "name" is always
    // the canonical modern id so consumers don't need to special-case old data.
    private static final String[] MODERN_ATTRIBUTE_IDS = {
            "minecraft:max_health",
            "minecraft:movement_speed",
            "minecraft:attack_damage",
            "minecraft:attack_knockback",
            "minecraft:attack_speed",
            "minecraft:armor",
            "minecraft:follow_range",
            "minecraft:knockback_resistance"
    };

    private static final String[] LEGACY_ATTRIBUTE_IDS = {
            "minecraft:generic.max_health",
            "minecraft:generic.movement_speed",
            "minecraft:generic.attack_damage",
            "minecraft:generic.attack_knockback",
            "minecraft:generic.attack_speed",
            "minecraft:generic.armor",
            "minecraft:generic.follow_range",
            "minecraft:generic.knockback_resistance"
    };

    private List<Map<String, Object>> extractAttributes(LivingEntity entity) {
        List<Map<String, Object>> attributes = new ArrayList<>();
        try {
            for (int index = 0; index < MODERN_ATTRIBUTE_IDS.length; index++) {
                String canonicalName = MODERN_ATTRIBUTE_IDS[index];
                String legacyName = LEGACY_ATTRIBUTE_IDS[index];
                try {
                    RegistryEntry<EntityAttribute> attrEntry = Registries.ATTRIBUTE
                            .getEntry(Identifier.splitOn(canonicalName, ':')).orElse(null);
                    if (attrEntry == null) {
                        attrEntry = Registries.ATTRIBUTE
                                .getEntry(Identifier.splitOn(legacyName, ':')).orElse(null);
                    }
                    if (attrEntry != null) {
                        double value = entity.getAttributeValue(attrEntry);
                        Map<String, Object> attrInfo = new LinkedHashMap<>();
                        attrInfo.put("name", canonicalName);
                        attrInfo.put("base_value", value);
                        attributes.add(attrInfo);
                    }
                } catch (Exception e) {
                    // Skip if attribute doesn't exist for this entity
                }
            }
        } catch (Exception e) {
            LOGGER.info("Could not extract attributes", e);
        }
        // Sort by name so unchanged data regenerates byte-identical rather than
        // depending on MODERN_ATTRIBUTE_IDS's declaration order.
        attributes.sort(Comparator.comparing(attr -> (String) attr.get("name")));
        return attributes;
    }

    private List<Map<String, Object>> extractSizeVariants(Entity entity) {
        List<Map<String, Object>> variants = new ArrayList<>();
        try {
            if (entity instanceof SlimeEntity slimeEntity) {
                for (int size = 1; size <= 4; size++) {
                    try {
                        slimeEntity.setSize(size, false);
                        Map<String, Object> sizeInfo = new LinkedHashMap<>();
                        sizeInfo.put("size", size);
                        sizeInfo.put("dimensions",
                                serializeEntityDimensions(slimeEntity.getDimensions(EntityPose.STANDING)));
                        variants.add(sizeInfo);
                    } catch (Exception e) {
                        // Skip if size setting fails
                    }
                }
            }
        } catch (Exception e) {
            LOGGER.debug("Could not extract size variants", e);
        }
        return variants;
    }

    private Map<String, Object> serializeEntityDimensions(net.minecraft.entity.EntityDimensions dimensions) {
        Map<String, Object> out = new LinkedHashMap<>();
        out.put("width", dimensions.width());
        out.put("height", dimensions.height());
        out.put("eye_height", dimensions.eyeHeight());
        out.put("fixed", dimensions.fixed());
        return out;
    }

    private Map<String, Object> collectItemInfo(Identifier id, Item item, MinecraftServer server)
            throws IllegalAccessException {
        Map<String, Object> info = new LinkedHashMap<>();

        info.put("id", id.toString());
        info.put("max_stack_size", item.getMaxCount());
        info.put("translation_key", item.getTranslationKey());

        // You can inspect use duration & animation via a default stack:
        ItemStack stack = new ItemStack(item);

        Rarity rarity = stack.get(DataComponentTypes.RARITY);
        if (rarity != null) {
            info.put("rarity", rarity.name());
        }

        // var fireResistant = stack.get(DataComponentTypes.FIRE_RESISTANT);
        // if (fireResistant != null) {
        // info.put("fireproof", true);
        // }

        var field = hasField("FIRE_RESISTANT", DataComponentTypes.class);
        if (field != null) {
            @SuppressWarnings("unchecked")
            ComponentType<Unit> dcType = (ComponentType<Unit>) field.get(null);
            var fireResistant = stack.get(dcType);
            if (fireResistant != null) {
                info.put("fireproof", true);
            } else {
                info.put("fireproof", false);
            }
        } else {
            info.put("fireproof", false);
        }

        info.put("use_animation", item.getUseAction(stack).name());

        List<String> tags = getItemTags(item);
        Map<String, Object> itemComponents = getItemComponents(item);

        info.put("tags", tags);
        info.put("components", itemComponents);

        info.put("is_weapon", isWeapon(tags, itemComponents));
        info.put("is_food", isFood(itemComponents));

        return info;
    }

    private Field hasField(String fieldName, Object obj) {
        try {
            Field field = obj.getClass().getDeclaredField("fieldName");
            // Field exists
            return field;
        } catch (NoSuchFieldException e) {
            // Field does not exist
            return null;
        }
    }

    private List<String> getItemTags(Item item) {
        List<String> tags = new ArrayList<>();
        RegistryEntry<Item> entry = Registries.ITEM.getEntry(item);

        entry.streamTags().forEach(tagKey -> {
            tags.add(tagKey.id().toString());
        });

        // streamTags() order isn't guaranteed stable; sort so unchanged data
        // regenerates byte-identical (matches the entity tags convention below).
        Collections.sort(tags);
        return tags;
    }

    private Map<String, Object> getItemComponents(Item item) {
        Map<String, Object> components = new LinkedHashMap<>();

        ItemStack stack = new ItemStack(item);
        var container = stack.getComponents(); // depends on mappings

        // Pseudocode-ish; mappings differ slightly:
        // Food component
        var food = stack.get(DataComponentTypes.FOOD);
        if (food != null) {
            Map<String, Object> foodInfo = new LinkedHashMap<>();
            foodInfo.put("nutrition", food.nutrition());
            foodInfo.put("saturation", food.saturation());
            foodInfo.put("can_always_eat", food.canAlwaysEat());
            components.put("food", foodInfo);
        }

        // DAMAGE/MAX_DAMAGE component exists only for damageable items
        Integer maxDamage = container.get(DataComponentTypes.MAX_DAMAGE);
        Integer maxDamageStack = stack.get(DataComponentTypes.MAX_DAMAGE);
        if (maxDamage != null) {
            components.put("max_damage", maxDamage);
        }

        if (maxDamageStack != null) {
            components.put("max_damage_stack", maxDamageStack);
        }

        // Damage
        var damage = stack.get(DataComponentTypes.DAMAGE);
        if (damage != null) {
            components.put("damage", damage.intValue()); // mapping-dependent
        }

        // Enchantability, repair item, etc., if you care:
        var enchant = stack.get(DataComponentTypes.ENCHANTMENTS);
        if (enchant != null) {
            components.put("enchantments", enchant.getEnchantments());
        }
        var tool = stack.get(DataComponentTypes.TOOL);
        if (tool != null) {
            components.put("is_tool", true);
        }
        return components;
    }

    private boolean isWeapon(List<String> tags, Map<String, Object> components) {
        // simple heuristics:
        return tags.contains("minecraft:swords") ||
                tags.contains("minecraft:axes") ||
                tags.contains("minecraft:weapons");
    }

    private boolean isFood(Map<String, Object> components) {
        return components.containsKey("food");
    }

    private JsonObject serializeProperties(BlockState state) {
        JsonObject obj = new JsonObject();
        for (Map.Entry<Property<?>, Comparable<?>> e : state.getEntries().entrySet()) {
            Property<?> property = e.getKey();
            Comparable<?> value = e.getValue();

            @SuppressWarnings({ "rawtypes", "unchecked" })
            String valueName = ((Property) property).name((Comparable) value);

            obj.addProperty(property.getName(), valueName);
        }
        return obj;
    }

    private JsonArray serializeShape(VoxelShape shape) {
        JsonArray arr = new JsonArray();
        List<Box> boxes = shape.getBoundingBoxes();
        for (Box box : boxes) {
            JsonObject o = new JsonObject();
            JsonArray min = new JsonArray();
            JsonArray max = new JsonArray();

            min.add(box.minX);
            min.add(box.minY);
            min.add(box.minZ);

            max.add(box.maxX);
            max.add(box.maxY);
            max.add(box.maxZ);

            o.add("min", min);
            o.add("max", max);
            arr.add(o);
        }
        return arr;
    }
}
