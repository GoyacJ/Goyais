/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: In-memory bootstrap command service for API and contract iteration.
 */
package com.ysmjjsy.goyais.application.command;

import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.CommandRef;
import com.ysmjjsy.goyais.contract.api.common.CommandResource;
import com.ysmjjsy.goyais.contract.api.common.ResourceBase;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.contract.api.common.WriteResponse;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ConcurrentMap;
import org.springframework.stereotype.Service;

/**
 * Exposes command create/list/get use cases for adapter layer.
 */
@Service
public final class CommandApplicationService {
    private final ConcurrentMap<String, CommandResource> commands = new ConcurrentHashMap<>();
    private final CommandPipeline pipeline;

    /**
     * Creates application service with command pipeline dependency.
     */
    public CommandApplicationService(CommandPipeline pipeline) {
        this.pipeline = pipeline;
    }

    /**
     * Creates command resource, executes pipeline and returns write response envelope.
     */
    public WriteResponse<CommandResource> create(CommandCreateRequest request, ExecutionContext context) {
        String id = UUID.randomUUID().toString();
        Instant now = Instant.now();

        Map<String, Object> result = pipeline.run(request, context);

        ResourceBase base = new ResourceBase(
                id,
                context.tenantId(),
                context.workspaceId(),
                context.userId(),
                request.visibility() == null ? Visibility.PRIVATE : request.visibility(),
                List.of(),
                "succeeded",
                now,
                now
        );

        CommandResource command = new CommandResource(
                base,
                request.commandType(),
                request.payload(),
                now,
                context.traceId(),
                result,
                null
        );

        commands.put(id, command);

        return new WriteResponse<>(command, new CommandRef(id, "accepted", now));
    }

    /**
     * Lists current in-memory command resources with newest-first ordering.
     */
    public List<CommandResource> list() {
        List<CommandResource> result = new ArrayList<>(commands.values());
        result.sort(Comparator.comparing(CommandResource::acceptedAt).reversed());
        return result;
    }

    /**
     * Returns one command resource by identifier.
     */
    public CommandResource get(String commandId) {
        return commands.get(commandId);
    }
}
