/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Canonical command endpoints for command-first bootstrap.
 */
package com.ysmjjsy.goyais.adapter.rest;

import com.ysmjjsy.goyais.application.command.CommandApplicationService;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.CommandResource;
import com.ysmjjsy.goyais.contract.api.common.ErrorEnvelope;
import com.ysmjjsy.goyais.contract.api.common.WriteResponse;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import jakarta.servlet.http.HttpServletRequest;
import java.util.List;
import java.util.Map;
import org.springframework.security.core.Authentication;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/v1/commands")
public final class CommandController {
    private final CommandApplicationService commandService;
    private final RequestExecutionContextFactory executionContextFactory;

    /**
     * Creates command controller with application service dependency.
     */
    public CommandController(
            CommandApplicationService commandService,
            RequestExecutionContextFactory executionContextFactory
    ) {
        this.commandService = commandService;
        this.executionContextFactory = executionContextFactory;
    }

    /**
     * Accepts canonical command create requests and returns command reference envelope.
     */
    @PostMapping
    public ResponseEntity<WriteResponse<CommandResource>> create(
            Authentication authentication,
            HttpServletRequest servletRequest,
            @RequestBody CommandCreateRequest request
    ) {
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        return ResponseEntity.accepted().body(commandService.create(request, context));
    }

    /**
     * Returns command list response compatible with Go pagination envelope semantics.
     */
    @GetMapping
    public Map<String, Object> list(Authentication authentication, HttpServletRequest servletRequest) {
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        List<CommandResource> items = commandService.list(context);
        return Map.of(
                "items", items,
                "pageInfo", Map.of("page", 1, "pageSize", items.size(), "total", items.size())
        );
    }

    /**
     * Returns one command resource by ID or contract error envelope when not found.
     */
    @GetMapping("/{commandId}")
    public ResponseEntity<?> get(
            @PathVariable String commandId,
            Authentication authentication,
            HttpServletRequest servletRequest
    ) {
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        CommandResource command = commandService.get(commandId, context);
        if (command == null) {
            return ResponseEntity.status(404).body(ErrorEnvelope.of("COMMAND_NOT_FOUND", "error.command.not_found"));
        }
        return ResponseEntity.ok(command);
    }
}
