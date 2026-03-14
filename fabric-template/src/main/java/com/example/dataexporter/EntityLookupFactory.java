package com.example.dataexporter;

import net.fabricmc.loader.api.FabricLoader;
import net.fabricmc.loader.api.Version;
// import net.fabricmc.loader.api.metadata.CustomValue;
import net.minecraft.entity.Entity;
import net.minecraft.entity.EntityType;
import net.minecraft.entity.SpawnReason;
import net.minecraft.server.world.ServerWorld;
import net.minecraft.util.math.BlockPos;
import java.lang.reflect.Method;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class EntityLookupFactory {
    private static final Logger LOGGER = LoggerFactory.getLogger("data_exporter");
    private static final boolean IS_MODERN = isVersionAtLeast("1.21.2");

    public static Entity createEntity(EntityType<?> type, ServerWorld world) {
        try {
            if (!IS_MODERN) {
                // 1.21.1 Logic: create(World)
                Method legacyCreate = EntityType.class.getMethod("create", net.minecraft.world.World.class);
                Entity entity = (Entity) legacyCreate.invoke(type, world);
                if (entity == null) {
                    LOGGER.debug("[EntityLookupFactory] create(World) returned null for {}", type);
                }
                return entity;
            } else {
                // 1.21.2+ Logic: create(ServerWorld, Nbt, Text, Player, BlockPos, SpawnReason,
                // bool, bool)
                // We use Reflection to avoid compile-time errors on 1.21.1
                for (Method m : EntityType.class.getMethods()) {
                    if (m.getName().equals("create") && m.getParameterCount() == 8) {
                        Entity entity = (Entity) m.invoke(type, world, null, null, null, BlockPos.ORIGIN, SpawnReason.TRIGGERED,
                                false, false);
                        if (entity == null) {
                            LOGGER.debug("[EntityLookupFactory] create(8 params) returned null for {}", type);
                        }
                        return entity;
                    }
                }
                LOGGER.debug("[EntityLookupFactory] No compatible create method found for {}", type);
            }
        } catch (Exception e) {
            LOGGER.debug("[EntityLookupFactory] Failed to instantiate {} via reflection: {}", type, e.toString());
        }
        return null;
    }

    private static boolean isVersionAtLeast(String versionStr) {
        return FabricLoader.getInstance().getModContainer("minecraft")
                .map(container -> {
                    Version current = container.getMetadata().getVersion();
                    try {
                        return current.compareTo(Version.parse(versionStr)) >= 0;
                    } catch (Exception e) {
                        return false;
                    }
                }).orElse(false);
    }
}
