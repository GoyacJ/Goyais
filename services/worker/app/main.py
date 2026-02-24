import logging
import os

from fastapi import FastAPI, Request

from app.orchestrator.claim_loop import ClaimLoopService
from app.routes import router
from app.trace import (
    TRACE_HEADER,
    generate_trace_id,
    get_trace_id,
    reset_trace_id,
    set_trace_id,
)


class TraceIDFilter(logging.Filter):
    def filter(self, record: logging.LogRecord) -> bool:
        record.trace_id = get_trace_id() or "-"
        return True


def configure_logging() -> logging.Logger:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s trace_id=%(trace_id)s %(message)s",
    )
    logger = logging.getLogger("goyais.worker")
    logger.setLevel(logging.INFO)
    trace_filter = TraceIDFilter()

    root_logger = logging.getLogger()
    for handler in root_logger.handlers:
        handler.addFilter(trace_filter)

    return logger


logger = configure_logging()
app = FastAPI(title="Goyais Worker", version="0.4.0")
claim_loop_service = ClaimLoopService()


@app.middleware("http")
async def trace_middleware(request: Request, call_next):
    trace_id = request.headers.get(TRACE_HEADER, "").strip() or generate_trace_id()
    request.state.trace_id = trace_id

    token = set_trace_id(trace_id)
    logger.info("request %s %s", request.method, request.url.path)
    try:
        response = await call_next(request)
    finally:
        reset_trace_id(token)

    response.headers[TRACE_HEADER] = trace_id
    return response


app.include_router(router)


@app.on_event("startup")
async def startup_worker_claim_loop() -> None:
    if os.getenv("WORKER_DISABLE_CLAIM_LOOP", "").strip().lower() in {"1", "true", "yes"}:
        return
    await claim_loop_service.start()


@app.on_event("shutdown")
async def shutdown_worker_claim_loop() -> None:
    if os.getenv("WORKER_DISABLE_CLAIM_LOOP", "").strip().lower() in {"1", "true", "yes"}:
        return
    await claim_loop_service.stop()


def get_port() -> int:
    port_raw = os.getenv("PORT", "8788").strip()
    try:
        return int(port_raw)
    except ValueError:
        return 8788


if __name__ == "__main__":
    import uvicorn

    uvicorn.run("app.main:app", host="127.0.0.1", port=get_port())
