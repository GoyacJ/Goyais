"""Auto-generated placeholder for protocol models.
Install datamodel-code-generator to regenerate strict models.
"""

from pydantic import BaseModel


class EventEnvelope(BaseModel):
  protocol_version: str
  trace_id: str
  event_id: str
  execution_id: str
  seq: int
  ts: str
  type: str
  payload: dict
