/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Bootstrap implementation of validate-authorize-execute-audit-event pipeline.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.command;

import com.ysmjjsy.goyais.application.audit.AuditEventStore;
import com.ysmjjsy.goyais.capability.event.DomainEvent;
import com.ysmjjsy.goyais.capability.event.DomainEventPublisher;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.domain.audit.AuditEvent;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import com.ysmjjsy.goyais.kernel.security.AuthorizationDecision;
import com.ysmjjsy.goyais.kernel.security.AuthorizationGate;
import com.ysmjjsy.goyais.kernel.security.EgressGate;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import org.springframework.core.annotation.AnnotationAwareOrderComparator;

/**
 * Implements validate-authorize-execute-audit-event pipeline for command-first flow.
 */
public final class CommandPipeline {
    private final AuthorizationGate authorizationGate;
    private final EgressGate egressGate;
    private final List<CommandHandler> handlers;
    private final DomainEventPublisher eventPublisher;
    private final AuditEventStore auditEventStore;

    /**
     * Creates pipeline with authorization, egress, command handlers and event publisher.
     * @param authorizationGate TODO
     * @param egressGate TODO
     * @param handlers TODO
     * @param eventPublisher TODO
     * @param auditEventStore TODO
     */
    public CommandPipeline(
            AuthorizationGate authorizationGate,
            EgressGate egressGate,
            List<CommandHandler> handlers,
            DomainEventPublisher eventPublisher,
            AuditEventStore auditEventStore
    ) {
        this.authorizationGate = authorizationGate;
        this.egressGate = egressGate;
        List<CommandHandler> sortedHandlers = new ArrayList<>(handlers);
        AnnotationAwareOrderComparator.sort(sortedHandlers);
        this.handlers = List.copyOf(sortedHandlers);
        this.eventPublisher = eventPublisher;
        this.auditEventStore = auditEventStore;
    }

    /**
     * Runs the full command pipeline and returns handler execution result.
     * @param request TODO
     * @param context TODO
     * @return TODO
     */
    public Map<String, Object> run(CommandCreateRequest request, ExecutionContext context) {
        validate(request);

        AuthorizationDecision authz = authorizationGate.authorize(request, context);
        recordAudit("command.authorize", context, request, authz.reason(), authz.allowed());
        if (!authz.allowed()) {
            throw new IllegalStateException("authorization denied: " + authz.reason());
        }

        AuthorizationDecision egress = egressGate.verify(request, context);
        recordAudit("command.egress", context, request, egress.reason(), egress.allowed());
        if (!egress.allowed()) {
            throw new IllegalStateException("egress denied: " + egress.reason());
        }

        Map<String, Object> result = handlers.stream()
                .filter(handler -> handler.supports(request.commandType()))
                .findFirst()
                .map(handler -> handler.execute(request, context))
                .orElseGet(() -> Map.of("status", "accepted", "note", "no command handler registered"));
        recordAudit("command.execute", context, request, "handler.executed", true);

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

    private void recordAudit(
            String type,
            ExecutionContext context,
            CommandCreateRequest request,
            String reason,
            boolean allowed
    ) {
        auditEventStore.save(new AuditEvent(
                type,
                context.traceId(),
                Instant.now(),
                Map.of(
                        "tenantId", context.tenantId(),
                        "workspaceId", context.workspaceId(),
                        "userId", context.userId(),
                        "commandType", request.commandType(),
                        "decision", allowed ? "allow" : "deny",
                        "reason", reason == null ? "" : reason,
                        "policyVersion", context.policyVersion()
                )
        ));
    }
}
