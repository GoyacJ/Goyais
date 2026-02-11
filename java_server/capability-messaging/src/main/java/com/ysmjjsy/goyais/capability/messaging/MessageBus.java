/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Message bus SPI with memory and kafka pluggable providers.
 */
package com.ysmjjsy.goyais.capability.messaging;

import com.ysmjjsy.goyais.capability.event.DomainEvent;
import java.util.function.Consumer;

public interface MessageBus {
    void publish(String topic, DomainEvent event);

    void subscribe(String topic, Consumer<DomainEvent> consumer);
}
