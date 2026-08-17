package com.example.dataexporter;

import net.fabricmc.loader.api.FabricLoader;
import net.fabricmc.loader.api.Version;
// import net.fabricmc.loader.api.metadata.CustomValue;
import net.minecraft.entity.Entity;
import net.minecraft.entity.EntityType;
import net.minecraft.entity.SpawnReason;
import net.minecraft.server.world.ServerWorld;
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
                // 1.21.2+ Logic: create(World, SpawnReason). The 8-parameter
                // create(ServerWorld, Nbt, Text, Player, BlockPos, SpawnReason, bool, bool)
                // this used to look for was never actually the right shape for any
                // 1.21.2+ version checked (1.21.2 through 1.21.11 only ever expose a
                // 6-param create(..., Consumer, ...) and this 2-param create(World,
                // SpawnReason) overload) — match on the SpawnReason parameter type
                // rather than a hardcoded parameter count to stay robust to future
                // signature churn.
                for (Method m : EntityType.class.getMethods()) {
                    if (m.getName().equals("create") && m.getParameterCount() == 2
                            && m.getParameterTypes()[1] == SpawnReason.class) {
                        Entity entity = (Entity) m.invoke(type, world, SpawnReason.TRIGGERED);
                        if (entity == null) {
                            LOGGER.debug("[EntityLookupFactory] create(World, SpawnReason) returned null for {}", type);
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
