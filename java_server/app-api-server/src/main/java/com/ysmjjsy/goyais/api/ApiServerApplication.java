/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Bootstrap entrypoint for Goyais Java API server.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.api;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * Bootstraps the unified API and authorization runtime entrypoint.
 */
@SpringBootApplication(scanBasePackages = "com.ysmjjsy.goyais")
public class ApiServerApplication {

    /**
     * Starts the single Java server application.
     * @param args TODO
     */
    public static void main(String[] args) {
        SpringApplication.run(ApiServerApplication.class, args);
    }
}
