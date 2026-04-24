#!/bin/sh
set -eu

cat >/usr/share/nginx/html/runtime-config.js <<EOF
window.__APP_CONFIG__ = {
  apiBaseUrl: "${API_BASE_URL:-}",
  firebaseApiKey: "${FIREBASE_API_KEY:-}",
  firebaseAuthDomain: "${FIREBASE_AUTH_DOMAIN:-}",
  firebaseProjectId: "${FIREBASE_PROJECT_ID:-}",
  firebaseAppId: "${FIREBASE_APP_ID:-}",
};
EOF
