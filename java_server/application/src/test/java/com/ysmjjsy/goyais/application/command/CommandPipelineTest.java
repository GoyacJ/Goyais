/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Unit tests for command pipeline authorization, egress, and audit behavior.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.command;

import com.ysmjjsy.goyais.application.audit.AuditEventStore;
import com.ysmjjsy.goyais.capability.event.DomainEvent;
import com.ysmjjsy.goyais.capability.event.DomainEventPublisher;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.domain.audit.AuditEvent;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import com.ysmjjsy.goyais.kernel.security.AuthorizationDecision;
import com.ysmjjsy.goyais.kernel.security.AuthorizationGate;
import com.ysmjjsy.goyais.kernel.security.EgressGate;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.Set;
import org.junit.jupiter.api.Assertions;
import org.junit.jupiter.api.Test;

class CommandPipelineTest {

    @Test
    void shouldRecordAuditAndExecuteWhenAuthorized() {
        List<AuditEvent> auditEvents = new ArrayList<>();
        List<DomainEvent> domainEvents = new ArrayList<>();

        AuthorizationGate authorizationGate = (request, context) -> AuthorizationDecision.allow("authz.ok");
        EgressGate egressGate = (request, context) -> AuthorizationDecision.allow("egress.ok");
        CommandHandler handler = new StaticCommandHandler();
        AuditEventStore auditEventStore = auditEvents::add;
        DomainEventPublisher eventPublisher = domainEvents::add;

        CommandPipeline pipeline = new CommandPipeline(
                authorizationGate,
                egressGate,
                List.of(handler),
                eventPublisher,
                auditEventStore
        );

        ExecutionContext context = executionContext();
        CommandCreateRequest request = new CommandCreateRequest("asset.create", Map.of("name", "demo"), Visibility.PRIVATE);

        Map<String, Object> result = pipeline.run(request, context);

        Assertions.assertEquals("executed", result.get("status"));
        Assertions.assertEquals(3, auditEvents.size());
        Assertions.assertEquals(1, domainEvents.size());
        Assertions.assertEquals("command.authorize", auditEvents.getFirst().type());
        Assertions.assertEquals("command.execute", auditEvents.getLast().type());
    }

    @Test
    void shouldStopAndAuditWhenAuthorizationDenied() {
        List<AuditEvent> auditEvents = new ArrayList<>();

        AuthorizationGate authorizationGate = (request, context) -> AuthorizationDecision.deny("rbac.blocked");
        EgressGate egressGate = (request, context) -> AuthorizationDecision.allow("egress.ok");
        CommandHandler handler = new StaticCommandHandler();
        AuditEventStore auditEventStore = auditEvents::add;
        DomainEventPublisher eventPublisher = event -> {
        };

        CommandPipeline pipeline = new CommandPipeline(
                authorizationGate,
                egressGate,
                List.of(handler),
                eventPublisher,
                auditEventStore
        );

        ExecutionContext context = executionContext();
        CommandCreateRequest request = new CommandCreateRequest("asset.create", Map.of(), Visibility.PRIVATE);

        IllegalStateException ex = Assertions.assertThrows(IllegalStateException.class, () -> pipeline.run(request, context));

        Assertions.assertTrue(ex.getMessage().contains("authorization denied"));
        Assertions.assertEquals(1, auditEvents.size());
        Assertions.assertEquals("command.authorize", auditEvents.getFirst().type());
        Assertions.assertEquals("deny", auditEvents.getFirst().payload().get("decision"));
    }

    private ExecutionContext executionContext() {
        return new ExecutionContext(
                "tenant-a",
                "workspace-a",
                "user-a",
                Set.of("member"),
                "v1",
                "trace-1"
        );
    }

    private static final class StaticCommandHandler implements CommandHandler {

        /**
         * Always accepts command types for pipeline unit testing.
         * @param commandType TODO
         * @return TODO
         */
        @Override
        public boolean supports(String commandType) {
            return true;
        }

        /**
         * Returns deterministic execution payload for assertions.
         * @param request TODO
         * @param context TODO
         * @return TODO
         */
        @Override
        public Map<String, Object> execute(CommandCreateRequest request, ExecutionContext context) {
            return Map.of("status", "executed", "commandType", request.commandType());
        }
    }
}
