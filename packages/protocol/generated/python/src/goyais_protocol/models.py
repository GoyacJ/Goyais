"""Auto-generated placeholder for protocol models.
Install datamodel-code-generator to regenerate strict models.
"""

from pydantic import BaseModel


class EventEnvelope(BaseModel):
  event_id: str
  run_id: str
  seq: int
  ts: str
  type: str
  payload: dict
