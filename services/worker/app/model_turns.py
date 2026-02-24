from __future__ import annotations

import asyncio
import json
from typing import Any
import urllib.error
import urllib.request

from app.model_google_payload import build_google_generate_url, to_google_payload
from app.model_adapters import (
    ModelAdapterError,
    ModelInvocation,
    ModelTurnResult,
    SUPPORTED_PARAM_KEYS,
    ToolCall,
)
from app.tls_config import TLSConfigError, resolve_tls_context


async def run_openai_compatible_turn(
    invocation: ModelInvocation, messages: list[dict[str, Any]], tools: list[dict[str, Any]]
) -> ModelTurnResult:
    url = f"{invocation.base_url}/chat/completions"
    payload: dict[str, Any] = {
        "model": invocation.model_id,
        "messages": _to_openai_messages(messages),
    }
    if tools:
        payload["tools"] = _to_openai_tools(tools)
    for key in SUPPORTED_PARAM_KEYS:
        if key in invocation.params:
            payload[key] = invocation.params[key]

    headers = {"Content-Type": "application/json"}
    if invocation.api_key:
        headers["Authorization"] = f"Bearer {invocation.api_key}"

    response = await _post_json(url, payload, headers, invocation.timeout_ms)
    choices = response.get("choices")
    if not isinstance(choices, list) or len(choices) == 0:
        raise ModelAdapterError("MODEL_EMPTY_RESPONSE", "OpenAI-compatible response has no choices")

    first_choice = choices[0]
    if not isinstance(first_choice, dict):
        raise ModelAdapterError("MODEL_INVALID_RESPONSE", "OpenAI-compatible choice must be an object")

    message = first_choice.get("message")
    message_map = message if isinstance(message, dict) else {}
    text = _extract_openai_text(message_map.get("content"))
    tool_calls = _extract_openai_tool_calls(message_map.get("tool_calls"))
    return ModelTurnResult(
        text=text,
        tool_calls=tool_calls,
        raw_response=response,
        usage=_extract_openai_usage(response.get("usage")),
    )


async def run_google_turn(
    invocation: ModelInvocation, messages: list[dict[str, Any]], tools: list[dict[str, Any]]
) -> ModelTurnResult:
    url = build_google_generate_url(invocation.base_url, invocation.model_id, invocation.api_key)
    payload = to_google_payload(messages, tools, invocation.params)
    headers = {"Content-Type": "application/json"}
    response = await _post_json(url, payload, headers, invocation.timeout_ms)

    candidates = response.get("candidates")
    if not isinstance(candidates, list) or len(candidates) == 0:
        raise ModelAdapterError("MODEL_EMPTY_RESPONSE", "Google response has no candidates", response)

    first_candidate = candidates[0]
    if not isinstance(first_candidate, dict):
        raise ModelAdapterError("MODEL_INVALID_RESPONSE", "Google candidate must be an object")

    content = first_candidate.get("content")
    content_map = content if isinstance(content, dict) else {}
    parts = content_map.get("parts")
    if not isinstance(parts, list):
        raise ModelAdapterError("MODEL_INVALID_RESPONSE", "Google candidate content.parts must be a list")

    text_fragments: list[str] = []
    tool_calls: list[ToolCall] = []
    for idx, part in enumerate(parts):
        if not isinstance(part, dict):
            continue
        text = str(part.get("text") or "").strip()
        if text:
            text_fragments.append(text)

        function_call = part.get("functionCall")
        if not isinstance(function_call, dict):
            function_call = part.get("function_call")
        if isinstance(function_call, dict):
            name = str(function_call.get("name") or "").strip()
            if name == "":
                continue
            args = function_call.get("args")
            tool_calls.append(
                ToolCall(
                    id=f"google_call_{idx + 1}",
                    name=name,
                    arguments=args if isinstance(args, dict) else {},
                )
            )

    return ModelTurnResult(
        text="\n".join(text_fragments).strip(),
        tool_calls=tool_calls,
        raw_response=response,
        usage=_extract_google_usage(response.get("usageMetadata")),
    )


async def _post_json(
    url: str, payload: dict[str, Any], headers: dict[str, str], timeout_ms: int
) -> dict[str, Any]:
    return await asyncio.to_thread(_post_json_sync, url, payload, headers, timeout_ms)


def _post_json_sync(
    url: str, payload: dict[str, Any], headers: dict[str, str], timeout_ms: int
) -> dict[str, Any]:
    body = json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(url, data=body, method="POST", headers=headers)
    timeout_seconds = max(timeout_ms / 1000.0, 1.0)
    try:
        context = resolve_tls_context(url)
    except TLSConfigError as exc:
        raise ModelAdapterError("MODEL_TLS_CONFIG_INVALID", str(exc), exc.details) from exc
    try:
        if context is None:
            response_context = urllib.request.urlopen(request, timeout=timeout_seconds)
        else:
            response_context = urllib.request.urlopen(request, timeout=timeout_seconds, context=context)
        with response_context as response:
            raw = response.read()
    except urllib.error.HTTPError as exc:
        raw = exc.read()
        raise ModelAdapterError(
            "MODEL_HTTP_ERROR",
            f"model request failed with status={exc.code}",
            {"status_code": exc.code, "body": _decode_error_body(raw)},
        ) from exc
    except urllib.error.URLError as exc:
        reason = exc.reason
        message = f"model request failed: {reason}"
        if isinstance(reason, BaseException) and reason.__class__.__name__ == "SSLError":
            message += (
                " (TLS 校验失败，可配置 WORKER_TLS_CA_FILE 指向企业 CA，"
                "或仅在受信环境下临时使用 WORKER_TLS_INSECURE_SKIP_VERIFY=1)"
            )
        raise ModelAdapterError(
            "MODEL_NETWORK_ERROR", message
        ) from exc

    try:
        parsed = json.loads(raw.decode("utf-8"))
    except ValueError as exc:
        raise ModelAdapterError("MODEL_INVALID_RESPONSE", "model response is not valid JSON") from exc
    if not isinstance(parsed, dict):
        raise ModelAdapterError("MODEL_INVALID_RESPONSE", "model response must be a JSON object")
    return parsed


def _to_openai_messages(messages: list[dict[str, Any]]) -> list[dict[str, Any]]:
    normalized: list[dict[str, Any]] = []
    for message in messages:
        role = str(message.get("role") or "").strip().lower()
        if role not in {"system", "user", "assistant", "tool"}:
            continue

        if role in {"system", "user"}:
            normalized.append({"role": role, "content": str(message.get("content") or "")})
            continue

        if role == "assistant":
            assistant: dict[str, Any] = {"role": "assistant"}
            content = str(message.get("content") or "").strip()
            if content:
                assistant["content"] = content
            raw_tool_calls = message.get("tool_calls")
            if isinstance(raw_tool_calls, list):
                assistant["tool_calls"] = _to_openai_assistant_tool_calls(raw_tool_calls)
            normalized.append(assistant)
            continue

        normalized.append(
            {
                "role": "tool",
                "tool_call_id": str(message.get("tool_call_id") or ""),
                "name": str(message.get("name") or ""),
                "content": str(message.get("content") or ""),
            }
        )
    return normalized


def _to_openai_assistant_tool_calls(raw_tool_calls: list[Any]) -> list[dict[str, Any]]:
    normalized_calls: list[dict[str, Any]] = []
    for item in raw_tool_calls:
        if not isinstance(item, dict):
            continue
        call_id = str(item.get("id") or "").strip() or "call_auto"
        name = str(item.get("name") or "").strip()
        if name == "":
            continue
        arguments = item.get("arguments")
        normalized_calls.append(
            {
                "id": call_id,
                "type": "function",
                "function": {
                    "name": name,
                    "arguments": json.dumps(arguments if isinstance(arguments, dict) else {}),
                },
            }
        )
    return normalized_calls


def _to_openai_tools(tools: list[dict[str, Any]]) -> list[dict[str, Any]]:
    normalized: list[dict[str, Any]] = []
    for tool in tools:
        name = str(tool.get("name") or "").strip()
        if name == "":
            continue
        normalized.append(
            {
                "type": "function",
                "function": {
                    "name": name,
                    "description": str(tool.get("description") or ""),
                    "parameters": tool.get("input_schema")
                    if isinstance(tool.get("input_schema"), dict)
                    else {"type": "object", "properties": {}},
                },
            }
        )
    return normalized


def _extract_openai_text(raw_content: Any) -> str:
    if isinstance(raw_content, str):
        return raw_content.strip()
    if isinstance(raw_content, list):
        text_parts: list[str] = []
        for item in raw_content:
            if isinstance(item, dict) and item.get("type") == "text":
                text = str(item.get("text") or "").strip()
                if text:
                    text_parts.append(text)
        return "\n".join(text_parts).strip()
    return ""


def _extract_openai_tool_calls(raw_tool_calls: Any) -> list[ToolCall]:
    if not isinstance(raw_tool_calls, list):
        return []
    tool_calls: list[ToolCall] = []
    for idx, item in enumerate(raw_tool_calls):
        if not isinstance(item, dict):
            continue
        function_block = item.get("function")
        if not isinstance(function_block, dict):
            continue
        name = str(function_block.get("name") or "").strip()
        if name == "":
            continue
        arguments = _parse_json_arguments(function_block.get("arguments"))
        tool_calls.append(
            ToolCall(
                id=str(item.get("id") or f"openai_call_{idx + 1}"),
                name=name,
                arguments=arguments,
            )
        )
    return tool_calls


def _parse_json_arguments(raw_arguments: Any) -> dict[str, Any]:
    if isinstance(raw_arguments, dict):
        return raw_arguments
    if not isinstance(raw_arguments, str):
        return {}
    try:
        parsed = json.loads(raw_arguments)
    except ValueError:
        return {}
    return parsed if isinstance(parsed, dict) else {}


def _decode_error_body(raw_body: bytes) -> str:
    try:
        parsed = json.loads(raw_body.decode("utf-8"))
        if isinstance(parsed, dict):
            return json.dumps(parsed)[:500]
    except ValueError:
        pass
    return raw_body.decode("utf-8", errors="ignore")[:500].strip()


def _extract_openai_usage(raw_usage: Any) -> dict[str, int]:
    usage = raw_usage if isinstance(raw_usage, dict) else {}
    input_tokens = _to_non_negative_int(usage.get("prompt_tokens"))
    output_tokens = _to_non_negative_int(usage.get("completion_tokens"))
    total_tokens = _to_non_negative_int(usage.get("total_tokens"))
    if total_tokens == 0:
        total_tokens = input_tokens + output_tokens
    return {
        "input_tokens": input_tokens,
        "output_tokens": output_tokens,
        "total_tokens": total_tokens,
    }


def _extract_google_usage(raw_usage: Any) -> dict[str, int]:
    usage = raw_usage if isinstance(raw_usage, dict) else {}
    input_tokens = _to_non_negative_int(usage.get("promptTokenCount"))
    output_tokens = _to_non_negative_int(usage.get("candidatesTokenCount"))
    total_tokens = _to_non_negative_int(usage.get("totalTokenCount"))
    if total_tokens == 0:
        total_tokens = input_tokens + output_tokens
    return {
        "input_tokens": input_tokens,
        "output_tokens": output_tokens,
        "total_tokens": total_tokens,
    }


def _to_non_negative_int(value: Any) -> int:
    try:
        parsed = int(value)
    except (TypeError, ValueError):
        return 0
    if parsed < 0:
        return 0
    return parsed
