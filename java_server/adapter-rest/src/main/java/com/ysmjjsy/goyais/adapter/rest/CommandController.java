/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Canonical command endpoints for command-first bootstrap.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
     * @param commandService TODO
     * @param executionContextFactory TODO
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
     * @param authentication TODO
     * @param servletRequest TODO
     * @param request TODO
     * @return TODO
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
     * @param authentication TODO
     * @param servletRequest TODO
     * @return TODO
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
     * @param commandId TODO
     * @param authentication TODO
     * @param servletRequest TODO
     * @return TODO
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
