package com.example.dataexporter;

import net.minecraft.world.entity.Entity;
import net.minecraft.world.entity.EntityType;
import net.minecraft.server.level.ServerLevel;
import java.lang.reflect.Method;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class EntityLookupFactory {
    private static final Logger LOGGER = LoggerFactory.getLogger("data_exporter");

    public static Entity createEntity(EntityType<?> type, ServerLevel level) {
        try {
            // Current signature: create(Level, EntitySpawnReason) — a 2-param
            // overload, not the 8-param create(ServerLevel, CompoundTag, Component,
            // Player, BlockPos, EntitySpawnReason, boolean, boolean) this used to
            // look for, which doesn't exist on 26.1 (confirmed against decompiled
            // EntityType.java). The spawn-reason enum's class/name has moved across
            // versions (EntitySpawnReason / MobSpawnType / SpawnReason), so it's
            // resolved by name via findSpawnReason() rather than a compile-time
            // import, and matched here purely on parameter count since it's the
            // only 2-param create(...) overload.
            Object spawnReason = findSpawnReason("TRIGGERED");
            for (Method method : EntityType.class.getMethods()) {
                if (method.getName().equals("create") && method.getParameterCount() == 2) {
                    Entity entity = (Entity) method.invoke(type, level, spawnReason);
                    if (entity == null) {
                        LOGGER.debug("[EntityLookupFactory] create(Level, SpawnReason) returned null for {}", type);
                    }
                    return entity;
                }
            }

            // Fallback: try create(ServerLevel) or create(Level)
            for (Method method : EntityType.class.getMethods()) {
                if (method.getName().equals("create") && method.getParameterCount() == 1) {
                    Entity entity = (Entity) method.invoke(type, level);
                    if (entity == null) {
                        LOGGER.debug("[EntityLookupFactory] create(1 param) returned null for {}", type);
                    }
                    return entity;
                }
            }

            LOGGER.debug("[EntityLookupFactory] No compatible create method found for {}", type);
        } catch (Exception exception) {
            LOGGER.debug("[EntityLookupFactory] Failed to instantiate {} via reflection: {}", type, exception.toString());
        }
        return null;
    }

    private static Object findSpawnReason(String valueName) {
        // Try known enum class names for spawn reason (varies between MC versions)
        String[] candidateClasses = {
                "net.minecraft.world.entity.EntitySpawnReason",
                "net.minecraft.world.entity.MobSpawnType",
                "net.minecraft.world.entity.SpawnReason"
        };
        for (String className : candidateClasses) {
            try {
                Class<?> enumClass = Class.forName(className);
                if (enumClass.isEnum()) {
                    for (Object constant : enumClass.getEnumConstants()) {
                        if (constant.toString().equals(valueName)) {
                            return constant;
                        }
                    }
                }
            } catch (ClassNotFoundException e) {
                // Try next candidate
            }
        }
        LOGGER.warn("[EntityLookupFactory] Could not find spawn reason enum value: {}", valueName);
        return null;
    }
}
