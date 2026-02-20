import json
from pathlib import Path

from jsonschema import Draft7Validator, RefResolver


def _load_validator() -> Draft7Validator:
    root = Path(__file__).resolve().parents[3]
    schema_dir = root / "packages" / "protocol" / "schemas" / "v2"

    schemas = {}
    for path in schema_dir.glob("*.json"):
        data = json.loads(path.read_text(encoding="utf-8"))
        schema_id = data.get("$id")
        if schema_id:
            schemas[schema_id] = data

    envelope = schemas["goyais.protocol.v2.event-envelope"]
    resolver = RefResolver.from_schema(envelope, store=schemas)
    return Draft7Validator(envelope, resolver=resolver)


def _assert_invalid(event: dict):
    validator = _load_validator()
    errors = list(validator.iter_errors(event))
    assert errors, "expected event to be invalid"


def test_tool_call_payload_requires_call_id():
    invalid_event = {
        "protocol_version": "2.0.0",
        "trace_id": "trace_1",
        "event_id": "evt_1",
        "run_id": "run_1",
        "seq": 1,
        "ts": "2026-02-20T00:00:00Z",
        "type": "tool_call",
        "payload": {"trace_id": "trace_1", "tool_name": "read_file", "args": {}, "requires_confirmation": False},
    }
    _assert_invalid(invalid_event)


def test_plan_payload_requires_summary():
    invalid_event = {
        "protocol_version": "2.0.0",
        "trace_id": "trace_1",
        "event_id": "evt_2",
        "run_id": "run_1",
        "seq": 2,
        "ts": "2026-02-20T00:00:00Z",
        "type": "plan",
        "payload": {"trace_id": "trace_1", "steps": ["read", "patch"]},
    }
    _assert_invalid(invalid_event)


def test_error_payload_requires_error():
    invalid_event = {
        "protocol_version": "2.0.0",
        "trace_id": "trace_1",
        "event_id": "evt_3",
        "run_id": "run_1",
        "seq": 3,
        "ts": "2026-02-20T00:00:00Z",
        "type": "error",
        "payload": {"trace_id": "trace_1", "message": "legacy"},
    }
    _assert_invalid(invalid_event)


def test_done_payload_requires_valid_status():
    invalid_event = {
        "protocol_version": "2.0.0",
        "trace_id": "trace_1",
        "event_id": "evt_4",
        "run_id": "run_1",
        "seq": 4,
        "ts": "2026-02-20T00:00:00Z",
        "type": "done",
        "payload": {"trace_id": "trace_1", "status": "ok"},
    }
    _assert_invalid(invalid_event)
