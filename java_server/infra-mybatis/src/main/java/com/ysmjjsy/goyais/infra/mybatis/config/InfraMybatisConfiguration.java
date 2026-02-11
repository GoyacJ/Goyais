/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>MyBatisPlus configuration for repository mapper scanning.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.infra.mybatis.config;

import com.baomidou.mybatisplus.extension.plugins.MybatisPlusInterceptor;
import org.mybatis.spring.annotation.MapperScan;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/**
 * Registers mapper scanning and baseline MyBatisPlus interceptors.
 */
@Configuration
@MapperScan("com.ysmjjsy.goyais.infra.mybatis.mapper")
public class InfraMybatisConfiguration {

    /**
     * Registers baseline MyBatisPlus interceptor chain.
     * @return TODO
     */
    @Bean
    public MybatisPlusInterceptor mybatisPlusInterceptor() {
        return new MybatisPlusInterceptor();
    }
}
