#!/bin/sh
set -e

# Ensure the file manager root is writable by the nonroot user.
# This is needed because named Docker volumes are created owned by root.
FM_ROOT="${FILE_MANAGER_ROOT:-/var/www}"
if [ -d "$FM_ROOT" ] && [ "$(stat -c '%u' "$FM_ROOT")" != "65532" ]; then
    chown -R 65532:65532 "$FM_ROOT" 2>/dev/null || true
fi

# Drop privileges and run the application.
exec su-exec nonroot /app/ventopanel "$@"
