/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Share REST controller aligned with Go-compatible /api/v1/shares routes.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.adapter.rest;

import com.ysmjjsy.goyais.application.share.ShareApplicationService;
import com.ysmjjsy.goyais.contract.api.common.Share;
import com.ysmjjsy.goyais.contract.api.common.ShareCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.ShareDeleteResult;
import com.ysmjjsy.goyais.contract.api.common.WriteResponse;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import jakarta.servlet.http.HttpServletRequest;
import java.util.List;
import java.util.Map;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

/**
 * Exposes share list and share command-sugar endpoints.
 */
@RestController
@RequestMapping("/api/v1/shares")
public final class ShareController {
    private final ShareApplicationService shareService;
    private final RequestExecutionContextFactory executionContextFactory;

    /**
     * Creates controller with share service and execution-context resolver.
     * @param shareService TODO
     * @param executionContextFactory TODO
     */
    public ShareController(
            ShareApplicationService shareService,
            RequestExecutionContextFactory executionContextFactory
    ) {
        this.shareService = shareService;
        this.executionContextFactory = executionContextFactory;
    }

    /**
     * Creates one share through command-first domain sugar endpoint.
     * @param authentication TODO
     * @param servletRequest TODO
     * @param request TODO
     * @return TODO
     */
    @PostMapping
    public ResponseEntity<WriteResponse<Share>> create(
            Authentication authentication,
            HttpServletRequest servletRequest,
            @RequestBody ShareCreateRequest request
    ) {
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        return ResponseEntity.accepted().body(shareService.create(request, context));
    }

    /**
     * Returns share list with pageInfo envelope.
     * @param authentication TODO
     * @param servletRequest TODO
     * @param page TODO
     * @param pageSize TODO
     * @return TODO
     */
    @GetMapping
    public Map<String, Object> list(
            Authentication authentication,
            HttpServletRequest servletRequest,
            @RequestParam(value = "page", defaultValue = "1") int page,
            @RequestParam(value = "pageSize", defaultValue = "20") int pageSize
    ) {
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        List<Share> items = shareService.list(context, page, pageSize);
        long total = shareService.count(context);

        return Map.of(
                "items", items,
                "pageInfo", Map.of(
                        "page", Math.max(page, 1),
                        "pageSize", Math.min(Math.max(pageSize, 1), 200),
                        "total", total
                )
        );
    }

    /**
     * Deletes one share through command-first domain sugar endpoint.
     * @param shareId TODO
     * @param authentication TODO
     * @param servletRequest TODO
     * @return TODO
     */
    @DeleteMapping("/{shareId}")
    public ResponseEntity<WriteResponse<ShareDeleteResult>> delete(
            @PathVariable String shareId,
            Authentication authentication,
            HttpServletRequest servletRequest
    ) {
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        return ResponseEntity.accepted().body(shareService.delete(shareId, context));
    }
}
