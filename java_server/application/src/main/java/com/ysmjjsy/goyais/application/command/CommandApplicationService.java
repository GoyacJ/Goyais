/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Command service backed by repository persistence for contract iteration.
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
import java.util.List;
import java.util.Map;
import java.util.UUID;
import org.springframework.stereotype.Service;

/**
 * Exposes command create/list/get use cases for adapter layer.
 */
@Service
public final class CommandApplicationService {
    private final CommandPipeline pipeline;
    private final CommandRepository commandRepository;

    /**
     * Creates application service with command pipeline dependency.
     */
    public CommandApplicationService(CommandPipeline pipeline, CommandRepository commandRepository) {
        this.pipeline = pipeline;
        this.commandRepository = commandRepository;
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

        commandRepository.save(command);

        return new WriteResponse<>(command, new CommandRef(id, "accepted", now));
    }

    /**
     * Lists readable command resources with newest-first ordering.
     */
    public List<CommandResource> list(ExecutionContext context) {
        return commandRepository.listReadable(context, 200);
    }

    /**
     * Returns one readable command resource by identifier.
     */
    public CommandResource get(String commandId, ExecutionContext context) {
        return commandRepository.findReadableById(commandId, context);
    }
}
