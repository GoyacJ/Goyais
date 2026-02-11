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
import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.UUID;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/v1/commands")
public final class CommandController {
    private final CommandApplicationService commandService;

    /**
     * Creates command controller with application service dependency.
     */
    public CommandController(CommandApplicationService commandService) {
        this.commandService = commandService;
    }

    /**
     * Accepts canonical command create requests and returns command reference envelope.
     */
    @PostMapping
    public ResponseEntity<WriteResponse<CommandResource>> create(
            @RequestHeader("X-Tenant-Id") String tenantId,
            @RequestHeader("X-Workspace-Id") String workspaceId,
            @RequestHeader("X-User-Id") String userId,
            @RequestHeader(value = "X-Roles", required = false, defaultValue = "member") String roles,
            @RequestHeader(value = "X-Policy-Version", required = false, defaultValue = "v0.1") String policyVersion,
            @RequestHeader(value = "X-Trace-Id", required = false) String traceId,
            @RequestBody CommandCreateRequest request
    ) {
        ExecutionContext context = new ExecutionContext(
                tenantId,
                workspaceId,
                userId,
                splitRoles(roles),
                policyVersion,
                traceId == null || traceId.isBlank() ? UUID.randomUUID().toString() : traceId
        );

        return ResponseEntity.accepted().body(commandService.create(request, context));
    }

    /**
     * Returns command list response compatible with Go pagination envelope semantics.
     */
    @GetMapping
    public Map<String, Object> list() {
        List<CommandResource> items = commandService.list();
        return Map.of(
                "items", items,
                "pageInfo", Map.of("page", 1, "pageSize", items.size(), "total", items.size())
        );
    }

    /**
     * Returns one command resource by ID or contract error envelope when not found.
     */
    @GetMapping("/{commandId}")
    public ResponseEntity<?> get(@PathVariable String commandId) {
        CommandResource command = commandService.get(commandId);
        if (command == null) {
            return ResponseEntity.status(404).body(ErrorEnvelope.of("COMMAND_NOT_FOUND", "error.command.not_found"));
        }
        return ResponseEntity.ok(command);
    }

    private Set<String> splitRoles(String roles) {
        return Arrays.stream(roles.split(","))
                .map(String::trim)
                .filter(s -> !s.isBlank())
                .collect(java.util.stream.Collectors.toSet());
    }
}
