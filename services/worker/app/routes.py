from fastapi import APIRouter, Request

from app.errors import standard_error_response

router = APIRouter()


@router.get("/health")
def health() -> dict[str, object]:
    return {"ok": True, "version": "0.4.0"}


@router.post("/internal/executions")
def internal_executions(request: Request):
    return standard_error_response(
        request=request,
        status_code=501,
        code="INTERNAL_NOT_IMPLEMENTED",
        message="Route is not implemented yet",
        details={"method": request.method, "path": request.url.path},
    )


@router.post("/internal/events")
def internal_events(request: Request):
    return standard_error_response(
        request=request,
        status_code=501,
        code="INTERNAL_NOT_IMPLEMENTED",
        message="Route is not implemented yet",
        details={"method": request.method, "path": request.url.path},
    )
