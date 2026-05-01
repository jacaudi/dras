"""Version constant.

The real version is injected by CI from the git tag at container-build time
via VERSION env var. For local development, this falls back to ``development``.
"""

import os

VERSION: str = os.environ.get("DRAS_RENDERER_VERSION", "development")
