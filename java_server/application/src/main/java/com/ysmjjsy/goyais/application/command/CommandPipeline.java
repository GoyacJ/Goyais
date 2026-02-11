/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Bootstrap implementation of validate-authorize-execute-audit-event pipeline.
 */
package com.ysmjjsy.goyais.application.command;

import com.ysmjjsy.goyais.capability.event.DomainEvent;
import com.ysmjjsy.goyais.capability.event.DomainEventPublisher;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import com.ysmjjsy.goyais.kernel.security.AuthorizationDecision;
import com.ysmjjsy.goyais.kernel.security.AuthorizationGate;
import com.ysmjjsy.goyais.kernel.security.EgressGate;
import java.time.Instant;
import java.util.List;
import java.util.Map;

public final class CommandPipeline {
    private final AuthorizationGate authorizationGate;
    private final EgressGate egressGate;
    private final List<CommandHandler> handlers;
    private final DomainEventPublisher eventPublisher;

    public CommandPipeline(
            AuthorizationGate authorizationGate,
            EgressGate egressGate,
            List<CommandHandler> handlers,
            DomainEventPublisher eventPublisher
    ) {
        this.authorizationGate = authorizationGate;
        this.egressGate = egressGate;
        this.handlers = handlers;
        this.eventPublisher = eventPublisher;
    }

    public Map<String, Object> run(CommandCreateRequest request, ExecutionContext context) {
        validate(request);

        AuthorizationDecision authz = authorizationGate.authorize(request, context);
        if (!authz.allowed()) {
            throw new IllegalStateException("authorization denied: " + authz.reason());
        }

        AuthorizationDecision egress = egressGate.verify(request, context);
        if (!egress.allowed()) {
            throw new IllegalStateException("egress denied: " + egress.reason());
        }

        Map<String, Object> result = handlers.stream()
                .filter(handler -> handler.supports(request.commandType()))
                .findFirst()
                .map(handler -> handler.execute(request, context))
                .orElseGet(() -> Map.of("status", "accepted", "note", "no command handler registered"));

        eventPublisher.publish(new DomainEvent(
                "command.execute",
                context.traceId(),
                Instant.now(),
                Map.of("commandType", request.commandType(), "status", "allow")
        ));

        return result;
    }

    private void validate(CommandCreateRequest request) {
        if (request == null || request.commandType() == null || request.commandType().isBlank()) {
            throw new IllegalArgumentException("commandType is required");
        }
    }
}
