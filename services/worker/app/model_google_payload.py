from __future__ import annotations

from typing import Any
import urllib.parse


def to_google_payload(
    messages: list[dict[str, Any]], tools: list[dict[str, Any]], params: dict[str, Any]
) -> dict[str, Any]:
    payload: dict[str, Any] = {"contents": _to_google_contents(messages)}
    system_instruction = _collect_system_instruction(messages)
    if system_instruction:
        payload["system_instruction"] = {"parts": [{"text": system_instruction}]}
    if tools:
        payload["tools"] = [{"functionDeclarations": _to_google_function_declarations(tools)}]

    generation_config = {
        key: params[key] for key in ("temperature", "top_p", "max_output_tokens") if key in params
    }
    if generation_config:
        payload["generationConfig"] = generation_config
    return payload


def build_google_generate_url(base_url: str, model_id: str, api_key: str) -> str:
    endpoint = f"{base_url.rstrip('/')}/models/{model_id}:generateContent"
    if api_key == "":
        return endpoint
    separator = "&" if urllib.parse.urlparse(endpoint).query else "?"
    return f"{endpoint}{separator}key={urllib.parse.quote(api_key)}"


def _to_google_contents(messages: list[dict[str, Any]]) -> list[dict[str, Any]]:
    contents: list[dict[str, Any]] = []
    for message in messages:
        role = str(message.get("role") or "").strip().lower()
        if role == "system":
            continue
        if role == "assistant":
            parts: list[dict[str, Any]] = []
            content = str(message.get("content") or "").strip()
            if content:
                parts.append({"text": content})
            raw_tool_calls = message.get("tool_calls")
            if isinstance(raw_tool_calls, list):
                for tool_call in raw_tool_calls:
                    if not isinstance(tool_call, dict):
                        continue
                    name = str(tool_call.get("name") or "").strip()
                    if name == "":
                        continue
                    args = tool_call.get("arguments")
                    parts.append(
                        {"functionCall": {"name": name, "args": args if isinstance(args, dict) else {}}}
                    )
            if parts:
                contents.append({"role": "model", "parts": parts})
            continue

        if role == "tool":
            name = str(message.get("name") or "").strip()
            if name == "":
                continue
            contents.append(
                {
                    "role": "user",
                    "parts": [
                        {
                            "functionResponse": {
                                "name": name,
                                "response": {"content": str(message.get("content") or "")},
                            }
                        }
                    ],
                }
            )
            continue

        text = str(message.get("content") or "").strip()
        if text:
            contents.append({"role": "user", "parts": [{"text": text}]})
    return contents


def _to_google_function_declarations(tools: list[dict[str, Any]]) -> list[dict[str, Any]]:
    declarations: list[dict[str, Any]] = []
    for tool in tools:
        name = str(tool.get("name") or "").strip()
        if name == "":
            continue
        declarations.append(
            {
                "name": name,
                "description": str(tool.get("description") or ""),
                "parameters": tool.get("input_schema")
                if isinstance(tool.get("input_schema"), dict)
                else {"type": "object", "properties": {}},
            }
        )
    return declarations


def _collect_system_instruction(messages: list[dict[str, Any]]) -> str:
    chunks: list[str] = []
    for message in messages:
        if str(message.get("role") or "").strip().lower() != "system":
            continue
        text = str(message.get("content") or "").strip()
        if text:
            chunks.append(text)
    return "\n".join(chunks).strip()
