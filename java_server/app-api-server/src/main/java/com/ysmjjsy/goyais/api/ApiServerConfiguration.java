/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Wiring defaults for command pipeline and security gates.
 */
package com.ysmjjsy.goyais.api;

import com.ysmjjsy.goyais.application.command.CommandHandler;
import com.ysmjjsy.goyais.application.command.CommandPipeline;
import com.ysmjjsy.goyais.capability.event.DomainEvent;
import com.ysmjjsy.goyais.capability.event.DomainEventPublisher;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import com.ysmjjsy.goyais.kernel.security.AuthorizationDecision;
import com.ysmjjsy.goyais.kernel.security.AuthorizationGate;
import com.ysmjjsy.goyais.kernel.security.EgressGate;
import java.util.List;
import java.util.Map;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class ApiServerConfiguration {

    @Bean
    public AuthorizationGate authorizationGate() {
        return (request, context) -> AuthorizationDecision.allow("bootstrap-allow");
    }

    @Bean
    public EgressGate egressGate() {
        return (request, context) -> AuthorizationDecision.allow("bootstrap-allow");
    }

    @Bean
    public DomainEventPublisher domainEventPublisher() {
        return event -> {
            // Bootstrap keeps in-process event sink for design-phase validation.
        };
    }

    @Bean
    public CommandHandler defaultCommandHandler() {
        return new CommandHandler() {
            @Override
            public boolean supports(String commandType) {
                return true;
            }

            @Override
            public Map<String, Object> execute(CommandCreateRequest request, ExecutionContext context) {
                return Map.of(
                        "commandType", request.commandType(),
                        "tenantId", context.tenantId(),
                        "workspaceId", context.workspaceId(),
                        "status", "executed"
                );
            }
        };
    }

    @Bean
    public CommandPipeline commandPipeline(
            AuthorizationGate authorizationGate,
            EgressGate egressGate,
            List<CommandHandler> handlers,
            DomainEventPublisher eventPublisher
    ) {
        return new CommandPipeline(authorizationGate, egressGate, handlers, eventPublisher);
    }
}
