package com.example.collisionexporter;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.google.gson.JsonArray;
import com.google.gson.JsonObject;
import net.fabricmc.api.ModInitializer;
import net.fabricmc.fabric.api.event.lifecycle.v1.ServerLifecycleEvents;

import net.minecraft.block.Block;
import net.minecraft.block.BlockState;
import net.minecraft.block.ShapeContext;
import net.minecraft.registry.Registries;
import net.minecraft.server.MinecraftServer;
import net.minecraft.state.property.Property;
import net.minecraft.util.Identifier;
import net.minecraft.util.math.BlockPos;
import net.minecraft.util.math.Box;
import net.minecraft.util.shape.VoxelShape;

import java.io.IOException;
import java.io.OutputStreamWriter;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
import java.util.Map;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class CollisionExporterMod implements ModInitializer {

    private static final Logger LOGGER = LoggerFactory.getLogger("collision_exporter");

    private static final Gson GSON = new GsonBuilder()
            .setPrettyPrinting()
            .disableHtmlEscaping()
            .create();

    @Override
    public void onInitialize() {
        LOGGER.info("[CollisionExporter] onInitialize");

        // Run once when the dedicated server has fully started.
        ServerLifecycleEvents.SERVER_STARTED.register(server -> {
            LOGGER.info("[CollisionExporter] SERVER_STARTED callback");
            try {
                dumpAllBlockStates(server);
            } catch (IOException e) {
                LOGGER.error("[CollisionExporter] Failed to dump data", e);
            } finally {
                LOGGER.info("[CollisionExporter] Stopping server after export");
                // In 1.21.x, stop(boolean waitForShutdown)
                server.stop(false);
            }
        });
    }

    private void dumpAllBlockStates(MinecraftServer server) throws IOException {
        Path runDir = server.getRunDirectory(); // This is a Path in 1.21.1+
        Path outDir = runDir.resolve("collision-data");
        Files.createDirectories(outDir);
        Path outFile = outDir.resolve("blocks.json");

        LOGGER.info("[CollisionExporter] Writing block data to {}", outFile.toAbsolutePath());

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

                allStates.add(entry);
            }
        }

        try (var writer = new OutputStreamWriter(
                Files.newOutputStream(outFile),
                StandardCharsets.UTF_8
        )) {
            GSON.toJson(allStates, writer);
        }

        LOGGER.info("[CollisionExporter] Finished writing {} blockstates", allStates.size());
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
