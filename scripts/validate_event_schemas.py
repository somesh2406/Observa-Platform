#!/usr/bin/env python3
"""Validate the Observa producer-event schema and representative contract cases.

Install development dependency with:
    python -m pip install -r requirements-dev.txt
Run from the repository root with:
    python scripts/validate_event_schemas.py
"""

from __future__ import annotations

import copy
import json
import sys
from pathlib import Path

from jsonschema import Draft202012Validator, FormatChecker
from referencing import Registry, Resource

ROOT = Path(__file__).resolve().parents[1]
SCHEMA_DIR = ROOT / "schemas"
EXAMPLE_PATH = ROOT / "examples" / "user-action.cloudevent.json"
SCHEMA_FILES = (
    "event-data-v1.schema.json",
    "cloudevent-v1.schema.json",
    "producer-event-v1.schema.json",
)


def load_json(path: Path):
    with path.open("r", encoding="utf-8") as file:
        return json.load(file)


def main() -> int:
    schemas = {name: load_json(SCHEMA_DIR / name) for name in SCHEMA_FILES}
    for name, schema in schemas.items():
        Draft202012Validator.check_schema(schema)
        if schema.get("$schema") != "https://json-schema.org/draft/2020-12/schema":
            raise AssertionError(f"{name}: expected JSON Schema Draft 2020-12")

    registry = Registry()
    for schema in schemas.values():
        registry = registry.with_resource(schema["$id"], Resource.from_contents(schema))

    producer_schema = schemas["producer-event-v1.schema.json"]
    validator = Draft202012Validator(
        producer_schema,
        registry=registry,
        format_checker=FormatChecker(),
    )
    corpus_path = ROOT / "tests" / "fixtures" / "event-contract-cases.json"
    cases = load_json(corpus_path)
    for case in cases:
        errors = sorted(validator.iter_errors(case["event"]), key=lambda error: list(error.path))
        valid = not errors
        if valid != case["valid"]:
            details = "; ".join(f"{'/'.join(map(str, error.path))}: {error.message}" for error in errors)
            expectation = "valid" if case["valid"] else "invalid"
            raise AssertionError(
                f"Case {case['name']!r} expected {expectation}, got "
                f"{'valid' if valid else 'invalid'}: {details}"
            )
        print(f"PASS: {'accepted' if valid else 'rejected'} {case['name']}")

    print(f"Validated {len(schemas)} schemas and {len(cases)} shared contract cases.")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # CLI should emit a concise actionable failure.
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
