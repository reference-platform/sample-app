import os

import uvicorn

from app.logs import setup_logging
from app.main import create_app

setup_logging()
uvicorn.run(
    create_app(),
    host="0.0.0.0",
    port=int(os.environ.get("PORT", "8000")),
    log_config=None,
)
