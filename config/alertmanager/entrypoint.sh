#!/bin/sh
set -e

# Substitute environment variables in alertmanager config template
# Using busybox sed because envsubst is not available in prom/alertmanager image
sed -e "s|\${TELEGRAM_BOT_TOKEN}|${TELEGRAM_BOT_TOKEN}|g" \
    -e "s|\${TELEGRAM_CHAT_ID}|${TELEGRAM_CHAT_ID}|g" \
    -e "s|\${TELEGRAM_MESSAGE_THREAD_ID}|${TELEGRAM_MESSAGE_THREAD_ID}|g" \
    -e "s|\${DISCORD_WEBHOOK_URL}|${DISCORD_WEBHOOK_URL}|g" \
    -e "s|\${SLACK_WEBHOOK_URL}|${SLACK_WEBHOOK_URL}|g" \
    /etc/alertmanager/alertmanager.yml.tmpl > /etc/alertmanager/alertmanager.yml

exec /bin/alertmanager "$@"
