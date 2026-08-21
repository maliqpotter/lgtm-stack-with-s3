#!/bin/sh
set -e

# Substitute environment variables in alertmanager config template
# Only substitute specific variables to avoid breaking Go template syntax ({{ .Alerts }}, etc.)
envsubst '${TELEGRAM_BOT_TOKEN} ${TELEGRAM_CHAT_ID} ${TELEGRAM_MESSAGE_THREAD_ID} ${DISCORD_WEBHOOK_URL} ${SLACK_WEBHOOK_URL}' \
  < /etc/alertmanager/alertmanager.yml.tmpl \
  > /etc/alertmanager/alertmanager.yml

exec /bin/alertmanager "$@"
