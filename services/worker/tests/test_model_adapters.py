import asyncio

import pytest

from app import model_turns
from app.model_adapters import (
    ModelAdapterError,
    ModelInvocation,
    resolve_model_invocation,
    run_model_turn,
)


def test_resolve_model_invocation_uses_snapshot_and_env_defaults() -> None:
    invocation = resolve_model_invocation(
        {
            "model_id": "gpt-4.1",
            "model_snapshot": {
                "vendor": "OpenAI",
                "model_id": "gpt-4.1",
                "params": {"temperature": 0.2},
            },
        },
        env={"OPENAI_API_KEY": "sk-test", "WORKER_MODEL_TIMEOUT_MS": "45000"},
    )

    assert invocation.vendor == "openai"
    assert invocation.model_id == "gpt-4.1"
    assert invocation.base_url == "https://api.openai.com/v1"
    assert invocation.api_key == "sk-test"
    assert invocation.timeout_ms == 45000
    assert invocation.params["temperature"] == 0.2


def test_resolve_model_invocation_google_requires_api_key() -> None:
    with pytest.raises(ModelAdapterError) as exc_info:
        resolve_model_invocation(
            {
                "model_id": "gemini-2.0-flash",
                "model_snapshot": {
                    "vendor": "Google",
                    "model_id": "gemini-2.0-flash",
                },
            },
            env={},
        )

    assert exc_info.value.code == "MODEL_API_KEY_MISSING"


def test_run_model_turn_openai_compatible_parses_tool_calls(monkeypatch: pytest.MonkeyPatch) -> None:
    captured: dict[str, object] = {}

    def fake_post_json_sync(url: str, payload: dict, headers: dict, timeout_ms: int) -> dict:
        captured["url"] = url
        captured["payload"] = payload
        captured["headers"] = headers
        captured["timeout_ms"] = timeout_ms
        return {
            "choices": [
                {
                    "message": {
                        "content": "ready",
                        "tool_calls": [
                            {
                                "id": "call_readme",
                                "type": "function",
                                "function": {
                                    "name": "read_file",
                                    "arguments": '{"path":"README.md"}',
                                },
                            }
                        ],
                    }
                }
            ]
        }

    monkeypatch.setattr(model_turns, "_post_json_sync", fake_post_json_sync)

    invocation = ModelInvocation(
        vendor="openai",
        model_id="gpt-4.1",
        base_url="https://api.openai.com/v1",
        api_key="sk-openai",
        timeout_ms=15000,
        params={"temperature": 0},
    )
    result = asyncio.run(
        run_model_turn(
            invocation=invocation,
            messages=[{"role": "user", "content": "read the repository summary"}],
            tools=[
                {
                    "name": "read_file",
                    "description": "Read one file",
                    "input_schema": {
                        "type": "object",
                        "properties": {"path": {"type": "string"}},
                    },
                }
            ],
        )
    )

    assert str(captured["url"]).endswith("/chat/completions")
    assert captured["payload"]["model"] == "gpt-4.1"  # type: ignore[index]
    assert captured["headers"]["Authorization"] == "Bearer sk-openai"  # type: ignore[index]
    assert result.text == "ready"
    assert len(result.tool_calls) == 1
    assert result.tool_calls[0].name == "read_file"
    assert result.tool_calls[0].arguments["path"] == "README.md"


def test_run_model_turn_google_parses_function_call(monkeypatch: pytest.MonkeyPatch) -> None:
    captured: dict[str, object] = {}

    def fake_post_json_sync(url: str, payload: dict, headers: dict, timeout_ms: int) -> dict:
        captured["url"] = url
        captured["payload"] = payload
        captured["headers"] = headers
        captured["timeout_ms"] = timeout_ms
        return {
            "candidates": [
                {
                    "content": {
                        "parts": [
                            {"text": "I will update the file."},
                            {
                                "functionCall": {
                                    "name": "write_file",
                                    "args": {"path": "src/main.ts", "content": "export {};"},
                                }
                            },
                        ]
                    }
                }
            ]
        }

    monkeypatch.setattr(model_turns, "_post_json_sync", fake_post_json_sync)

    invocation = ModelInvocation(
        vendor="google",
        model_id="gemini-2.0-flash",
        base_url="https://generativelanguage.googleapis.com/v1beta",
        api_key="google-key",
        timeout_ms=20000,
        params={},
    )
    result = asyncio.run(
        run_model_turn(
            invocation=invocation,
            messages=[{"role": "user", "content": "update main.ts"}],
            tools=[
                {
                    "name": "write_file",
                    "description": "Write one file",
                    "input_schema": {
                        "type": "object",
                        "properties": {
                            "path": {"type": "string"},
                            "content": {"type": "string"},
                        },
                    },
                }
            ],
        )
    )

    assert "models/gemini-2.0-flash:generateContent" in str(captured["url"])
    assert "key=google-key" in str(captured["url"])
    assert result.text == "I will update the file."
    assert len(result.tool_calls) == 1
    assert result.tool_calls[0].name == "write_file"
    assert result.tool_calls[0].arguments["path"] == "src/main.ts"
