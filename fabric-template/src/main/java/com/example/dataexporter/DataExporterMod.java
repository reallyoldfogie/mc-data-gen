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
import net.minecraft.item.Item;
import net.minecraft.item.ItemStack;
import net.minecraft.registry.Registries;
import net.minecraft.registry.entry.RegistryEntry;
import net.minecraft.registry.tag.BlockTags;
import net.minecraft.registry.tag.FluidTags;
import net.minecraft.server.MinecraftServer;
import net.minecraft.state.property.Property;
import net.minecraft.util.Identifier;
import net.minecraft.util.Rarity;
import net.minecraft.util.math.BlockPos;
import net.minecraft.util.math.Box;
import net.minecraft.util.shape.VoxelShape;
import net.minecraft.util.Unit;

import java.io.IOException;
import java.io.OutputStreamWriter;
import java.lang.reflect.Field;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

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

        JsonArray allStates = new JsonArray();

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

                allStates.add(entry);
            }
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

        Map<String, Object> allItems = new LinkedHashMap<>();
        // JsonArray allItems = new JsonArray();

        for (Identifier id : Registries.ITEM.getIds()) {
            Item item = Registries.ITEM.get(id);
            Map<String, Object> info = collectItemInfo(id, item, server);
            allItems.put(id.toString(), info);
        }

        try (var writer = new OutputStreamWriter(
                Files.newOutputStream(outFile),
                StandardCharsets.UTF_8)) {
            GSON.toJson(allItems, writer);
        }
    }

    private Map<String, Object> collectItemInfo(Identifier id, Item item, MinecraftServer server) throws IllegalAccessException {
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
