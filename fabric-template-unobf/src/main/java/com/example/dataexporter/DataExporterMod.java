package com.example.dataexporter;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonArray;
import com.google.gson.JsonObject;
import net.fabricmc.api.ModInitializer;
import net.fabricmc.fabric.api.event.lifecycle.v1.ServerLifecycleEvents;

import net.minecraft.core.BlockPos;
import net.minecraft.core.Holder;
import net.minecraft.core.registries.BuiltInRegistries;
import net.minecraft.resources.Identifier;
import net.minecraft.server.MinecraftServer;
import net.minecraft.server.level.ServerLevel;
import net.minecraft.server.level.ServerPlayer;
import net.minecraft.tags.BlockTags;
import net.minecraft.tags.FluidTags;
import net.minecraft.util.Unit;
import net.minecraft.world.entity.AgeableMob;
import net.minecraft.world.entity.Entity;
import net.minecraft.world.entity.EntityDimensions;
import net.minecraft.world.entity.EntityType;
import net.minecraft.world.entity.LivingEntity;
import net.minecraft.world.entity.Mob;
import net.minecraft.world.entity.Pose;
import net.minecraft.world.entity.ai.attributes.Attribute;
import net.minecraft.world.entity.ai.attributes.AttributeInstance;
import net.minecraft.world.entity.ai.attributes.Attributes;
import net.minecraft.world.entity.monster.Phantom;
import net.minecraft.world.entity.monster.Slime;
import net.minecraft.world.item.Item;
import net.minecraft.world.item.ItemStack;
import net.minecraft.world.item.Items;
import net.minecraft.world.item.Rarity;
import net.minecraft.world.level.block.Block;
import net.minecraft.world.level.block.state.BlockState;
import net.minecraft.world.level.block.state.properties.Property;
import net.minecraft.world.phys.AABB;
import net.minecraft.world.phys.shapes.CollisionContext;
import net.minecraft.world.phys.shapes.VoxelShape;
import net.minecraft.core.component.DataComponentType;
import net.minecraft.core.component.DataComponents;
import com.mojang.authlib.GameProfile;

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

        ServerLifecycleEvents.SERVER_STARTED.register(server -> {
            LOGGER.info("[DataExporter] SERVER_STARTED callback");
            try {
                dumpAllBlockStates(server);
                dumpItems(server);
                dumpEntities(server);
            } catch (IOException e) {
                LOGGER.error("[DataExporter] Failed to dump data", e);
            } catch (IllegalAccessException e) {
                LOGGER.error("[DataExporter] Failed to dump data", e);
            } finally {
                LOGGER.info("[DataExporter] Stopping server after export");
                server.halt(false);
            }
        });
    }

    private Map<String, Object> buildAttribute(String name, double baseValue) {
        Map<String, Object> attrInfo = new LinkedHashMap<>();
        attrInfo.put("name", name);
        attrInfo.put("base_value", baseValue);
        return attrInfo;
    }

    private void dumpAllBlockStates(MinecraftServer server) throws IOException {
        Path runDir = server.getServerDirectory();
        Path outDir = runDir.resolve("data");
        Files.createDirectories(outDir);
        Path outFile = outDir.resolve("blocks.json");

        LOGGER.info("[DataExporter] Writing block data to {}", outFile.toAbsolutePath());

        List<JsonObject> allEntries = new ArrayList<>();

        ServerLevel level = server.overworld();
        BlockPos pos = BlockPos.ZERO;
        CollisionContext ctx = CollisionContext.empty();

        for (Block block : BuiltInRegistries.BLOCK) {
            Identifier id = BuiltInRegistries.BLOCK.getKey(block);

            for (BlockState state : block.getStateDefinition().getPossibleStates()) {
                JsonObject entry = new JsonObject();
                entry.addProperty("block_id", id.toString());
                entry.add("properties", serializeProperties(state));

                VoxelShape collision = state.getCollisionShape(level, pos, ctx);
                entry.add("collision_boxes", serializeShape(collision));

                VoxelShape outline = state.getShape(level, pos, ctx);
                entry.add("outline_boxes", serializeShape(outline));

                entry.addProperty("air", state.isAir());
                entry.addProperty("opaque", state.canOcclude());
                entry.addProperty("solid_block", state.isRedstoneConductor(level, pos));
                entry.addProperty("replaceable", state.canBeReplaced());
                entry.addProperty("blocks_movement", !collision.isEmpty());

                boolean climbable = state.is(BlockTags.CLIMBABLE);
                entry.addProperty("climbable", climbable);

                boolean doorLike = state.is(BlockTags.DOORS)
                        || state.is(BlockTags.TRAPDOORS)
                        || state.is(BlockTags.FENCE_GATES);
                entry.addProperty("door_like", doorLike);

                boolean fenceLike = state.is(BlockTags.FENCES)
                        || state.is(BlockTags.WALLS);
                entry.addProperty("fence_like", fenceLike);

                boolean slab = state.is(BlockTags.SLABS);
                boolean stair = state.is(BlockTags.STAIRS);
                entry.addProperty("slab", slab);
                entry.addProperty("stair", stair);

                boolean logOrLeaf = state.is(BlockTags.LOGS)
                        || state.is(BlockTags.LEAVES);
                entry.addProperty("log_or_leaf", logOrLeaf);

                var fluidState = state.getFluidState();
                boolean isWater = fluidState.is(FluidTags.WATER);
                boolean isLava = fluidState.is(FluidTags.LAVA);
                boolean isFluid = !fluidState.isEmpty();

                entry.addProperty("water", isWater);
                entry.addProperty("lava", isLava);
                entry.addProperty("fluid", isFluid);

                float hardness = state.getDestroySpeed(level, pos);
                entry.addProperty("hardness", hardness);
                entry.addProperty("resistance", block.getExplosionResistance());

                Item blockItem = block.asItem();
                int stackSize = (blockItem != null && blockItem != Items.AIR)
                        ? blockItem.getDefaultMaxStackSize() : 0;
                entry.addProperty("stack_size", stackSize);

                entry.addProperty("diggable", hardness >= 0);

                JsonArray materialArray = new JsonArray();
                if (state.is(BlockTags.MINEABLE_WITH_PICKAXE)) materialArray.add("mineable/pickaxe");
                if (state.is(BlockTags.MINEABLE_WITH_AXE)) materialArray.add("mineable/axe");
                if (state.is(BlockTags.MINEABLE_WITH_SHOVEL)) materialArray.add("mineable/shovel");
                if (state.is(BlockTags.MINEABLE_WITH_HOE)) materialArray.add("mineable/hoe");
                entry.add("material", materialArray);

                allEntries.add(entry);
            }
        }

        allEntries.sort(Comparator.comparing(entry -> entry.get("block_id").getAsString()));

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
        Path runDir = server.getServerDirectory();
        Path outDir = runDir.resolve("data");
        Files.createDirectories(outDir);
        Path outFile = outDir.resolve("items.json");
        List<Map<String, Object>> allItems = new ArrayList<>();

        LOGGER.info("[DataExporter] Starting item export");
        for (Identifier id : BuiltInRegistries.ITEM.keySet()) {
            Item item = BuiltInRegistries.ITEM.getValue(id);
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

    private void dumpEntities(MinecraftServer server) throws IOException {
        Path runDir = server.getServerDirectory();
        Path outDir = runDir.resolve("data");
        Files.createDirectories(outDir);
        Path outFile = outDir.resolve("entities.json");

        ServerLevel level = server.overworld();
        List<Map<String, Object>> allEntities = new ArrayList<>();

        LOGGER.info("[DataExporter] Starting entity export");
        for (EntityType<?> entityType : BuiltInRegistries.ENTITY_TYPE) {
            Identifier id = BuiltInRegistries.ENTITY_TYPE.getKey(entityType);
            Map<String, Object> info = new LinkedHashMap<>();
            info.put("entity_id", id.toString());
            info.put("spawn_group", entityType.getCategory().name());
            info.put("fire_immune", entityType.fireImmune());

            var defaultDims = entityType.getDimensions();
            info.put("default_dimensions", serializeEntityDimensions(defaultDims));

            String entityId = id.toString();
            Entity entity = null;

            if (entityId.equals("minecraft:player")) {
                try {
                    GameProfile dummyProfile = new GameProfile(UUID.randomUUID(), "DataGenPlayer");
                    entity = new ServerPlayer(level.getServer(), level, dummyProfile, null);
                    LOGGER.info("[DataExporter] Entity instantiation OK: {}", id);
                } catch (Exception e) {
                    LOGGER.info("[DataExporter] Entity instantiation FAILED: {} - {}", id, e.getMessage());
                    entity = null;
                }
            } else {
                entity = EntityLookupFactory.createEntity(entityType, level);
                if (entity == null) {
                    LOGGER.info("[DataExporter] Entity instantiation FAILED: {}", id);
                } else {
                    LOGGER.info("[DataExporter] Entity instantiation OK: {}", id);
                }
            }

            Map<String, Object> poseDims = new TreeMap<>();
            if (entity != null) {
                for (Pose pose : Pose.values()) {
                    try {
                        var dims = entity.getDimensions(pose);
                        String poseName = pose.name().toLowerCase();

                        if (dims.width() == 0.2f && dims.height() == 0.2f && dims.eyeHeight() == 0.2f && dims.fixed()) {
                            continue;
                        }

                        if (dims.width() != defaultDims.width() || dims.height() != defaultDims.height()
                                || dims.eyeHeight() != defaultDims.eyeHeight()) {
                            poseDims.put(poseName, serializeEntityDimensions(dims));
                        }
                    } catch (Exception e) {
                        // Skip poses that don't exist for this entity
                    }
                }
            } else if (entityId.equals("minecraft:player")) {
                poseDims.put("standing", serializeEntityDimensions(defaultDims));
                Map<String, Object> crouchDims = new LinkedHashMap<>();
                crouchDims.put("width", 0.6);
                crouchDims.put("height", 1.5);
                crouchDims.put("eye_height", 1.27);
                crouchDims.put("fixed", false);
                poseDims.put("crouching", crouchDims);
                Map<String, Object> swimDims = new LinkedHashMap<>();
                swimDims.put("width", 0.6);
                swimDims.put("height", 0.6);
                swimDims.put("eye_height", 0.6);
                swimDims.put("fixed", false);
                poseDims.put("swimming", swimDims);
            }
            info.put("pose_dimensions", poseDims);

            try {
                if (entity != null) {
                    List<Map<String, Object>> attributes = Collections.emptyList();
                    if (entity instanceof LivingEntity livingEntity) {
                        attributes = extractAttributes(livingEntity);
                    }
                    info.put("attributes", attributes);

                    List<Map<String, Object>> sizeVariants = extractSizeVariants(entity);
                    info.put("size_variants", sizeVariants);

                    if (entity instanceof AgeableMob animal) {
                        animal.setBaby(true);
                        var babyDimensions = animal.getDimensions(Pose.STANDING);
                        if (babyDimensions != defaultDims) {
                            info.put("baby_dimensions",
                                    serializeEntityDimensions(babyDimensions));
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
                if (entityId.equals("minecraft:player") && info.get("attributes").equals(Collections.emptyList())) {
                    List<Map<String, Object>> playerAttributes = new ArrayList<>();
                    playerAttributes.add(buildAttribute("minecraft:generic.max_health", 20.0));
                    playerAttributes.add(buildAttribute("minecraft:generic.movement_speed", 0.1));
                    playerAttributes.add(buildAttribute("minecraft:generic.attack_damage", 1.0));
                    playerAttributes.add(buildAttribute("minecraft:generic.attack_speed", 4.0));
                    playerAttributes.add(buildAttribute("minecraft:generic.armor", 0.0));
                    playerAttributes.add(buildAttribute("minecraft:generic.attack_knockback", 0.0));
                    playerAttributes.add(buildAttribute("minecraft:generic.knockback_resistance", 0.0));
                    playerAttributes.add(buildAttribute("minecraft:generic.follow_range", 32.0));
                    info.put("attributes", playerAttributes);
                }

                if (entity != null) {
                    entity.discard();
                }
            }

            List<String> tags = new ArrayList<>();
            Holder<EntityType<?>> entry = BuiltInRegistries.ENTITY_TYPE.wrapAsHolder(entityType);
            entry.tags().forEach(tagKey -> tags.add(tagKey.location().toString()));
            Collections.sort(tags);
            info.put("tags", tags);

            allEntities.add(info);
        }

        allEntities.sort(Comparator.comparing(entry -> entry.get("entity_id").toString()));
        try (var writer = new OutputStreamWriter(Files.newOutputStream(outFile), StandardCharsets.UTF_8)) {
            GSON.toJson(allEntities, writer);
        }
        LOGGER.info("[DataExporter] Finished writing {} entities", allEntities.size());
    }

    private List<Map<String, Object>> extractAttributes(LivingEntity entity) {
        List<Map<String, Object>> attributes = new ArrayList<>();
        try {
            String[] commonAttrs = {
                    "minecraft:generic.max_health",
                    "minecraft:generic.movement_speed",
                    "minecraft:generic.attack_damage",
                    "minecraft:generic.attack_knockback",
                    "minecraft:generic.attack_speed",
                    "minecraft:generic.armor",
                    "minecraft:generic.follow_range",
                    "minecraft:generic.knockback_resistance"
            };

            for (String attrName : commonAttrs) {
                try {
                    Holder<Attribute> attrEntry = BuiltInRegistries.ATTRIBUTE
                            .get(Identifier.parse(attrName)).orElse(null);
                    if (attrEntry != null) {
                        double value = entity.getAttributeValue(attrEntry);
                        Map<String, Object> attrInfo = new LinkedHashMap<>();
                        attrInfo.put("name", attrName);
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
        return attributes;
    }

    private List<Map<String, Object>> extractSizeVariants(Entity entity) {
        List<Map<String, Object>> variants = new ArrayList<>();
        try {
            if (entity instanceof Slime slimeEntity) {
                for (int size = 1; size <= 4; size++) {
                    try {
                        slimeEntity.setSize(size, false);
                        Map<String, Object> sizeInfo = new LinkedHashMap<>();
                        sizeInfo.put("size", size);
                        sizeInfo.put("dimensions",
                                serializeEntityDimensions(slimeEntity.getDimensions(Pose.STANDING)));
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

    private Map<String, Object> serializeEntityDimensions(EntityDimensions dimensions) {
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
        info.put("max_stack_size", item.getDefaultMaxStackSize());
        info.put("translation_key", item.getDescriptionId());

        ItemStack stack = new ItemStack(item);

        Rarity rarity = stack.get(DataComponents.RARITY);
        if (rarity != null) {
            info.put("rarity", rarity.name());
        }

        var field = hasField("FIRE_RESISTANT", DataComponents.class);
        if (field != null) {
            @SuppressWarnings("unchecked")
            DataComponentType<Unit> dcType = (DataComponentType<Unit>) field.get(null);
            var fireResistant = stack.get(dcType);
            if (fireResistant != null) {
                info.put("fireproof", true);
            } else {
                info.put("fireproof", false);
            }
        } else {
            info.put("fireproof", false);
        }

        info.put("use_animation", item.getUseAnimation(stack).name());

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
            Field field = obj.getClass().getDeclaredField(fieldName);
            return field;
        } catch (NoSuchFieldException e) {
            return null;
        }
    }

    private List<String> getItemTags(Item item) {
        List<String> tags = new ArrayList<>();
        Holder<Item> entry = BuiltInRegistries.ITEM.wrapAsHolder(item);

        entry.tags().forEach(tagKey -> {
            tags.add(tagKey.location().toString());
        });

        return tags;
    }

    private Map<String, Object> getItemComponents(Item item) {
        Map<String, Object> components = new LinkedHashMap<>();

        ItemStack stack = new ItemStack(item);
        var container = stack.getComponents();

        var food = stack.get(DataComponents.FOOD);
        if (food != null) {
            Map<String, Object> foodInfo = new LinkedHashMap<>();
            foodInfo.put("nutrition", food.nutrition());
            foodInfo.put("saturation", food.saturation());
            foodInfo.put("can_always_eat", food.canAlwaysEat());
            components.put("food", foodInfo);
        }

        Integer maxDamage = container.get(DataComponents.MAX_DAMAGE);
        Integer maxDamageStack = stack.get(DataComponents.MAX_DAMAGE);
        if (maxDamage != null) {
            components.put("max_damage", maxDamage);
        }

        if (maxDamageStack != null) {
            components.put("max_damage_stack", maxDamageStack);
        }

        var damage = stack.get(DataComponents.DAMAGE);
        if (damage != null) {
            components.put("damage", damage.intValue());
        }

        var enchant = stack.get(DataComponents.ENCHANTMENTS);
        if (enchant != null) {
            components.put("enchantments", enchant.keySet());
        }
        var tool = stack.get(DataComponents.TOOL);
        if (tool != null) {
            components.put("is_tool", true);
        }
        return components;
    }

    private boolean isWeapon(List<String> tags, Map<String, Object> components) {
        return tags.contains("minecraft:swords") ||
                tags.contains("minecraft:axes") ||
                tags.contains("minecraft:weapons");
    }

    private boolean isFood(Map<String, Object> components) {
        return components.containsKey("food");
    }

    private JsonObject serializeProperties(BlockState state) {
        JsonObject obj = new JsonObject();
        // getValues() returns Stream<Property.Value<?>> in 26.1+
        state.getValues().forEach(propValue -> {
            obj.addProperty(propValue.property().getName(), propValue.valueName());
        });
        return obj;
    }

    private JsonArray serializeShape(VoxelShape shape) {
        JsonArray arr = new JsonArray();
        List<AABB> boxes = shape.toAabbs();
        for (AABB box : boxes) {
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
