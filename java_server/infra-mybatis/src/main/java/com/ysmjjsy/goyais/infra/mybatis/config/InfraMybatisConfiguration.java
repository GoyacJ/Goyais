/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: MyBatisPlus configuration for repository mapper scanning.
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
     */
    @Bean
    public MybatisPlusInterceptor mybatisPlusInterceptor() {
        return new MybatisPlusInterceptor();
    }
}
