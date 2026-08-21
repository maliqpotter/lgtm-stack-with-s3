#!/bin/sh
set -e

# Copy template to final config
cp /etc/alertmanager/alertmanager.yml.tmpl /etc/alertmanager/alertmanager.yml

# Substitute environment variables using sed (envsubst not available in this image)
sed -i "s|\${TELEGRAM_BOT_TOKEN}|${TELEGRAM_BOT_TOKEN}|g" /etc/alertmanager/alertmanager.yml
sed -i "s|\${TELEGRAM_CHAT_ID}|${TELEGRAM_CHAT_ID}|g" /etc/alertmanager/alertmanager.yml
sed -i "s|\${TELEGRAM_MESSAGE_THREAD_ID}|${TELEGRAM_MESSAGE_THREAD_ID}|g" /etc/alertmanager/alertmanager.yml
sed -i "s|\${DISCORD_WEBHOOK_URL}|${DISCORD_WEBHOOK_URL}|g" /etc/alertmanager/alertmanager.yml
sed -i "s|\${SLACK_WEBHOOK_URL}|${SLACK_WEBHOOK_URL}|g" /etc/alertmanager/alertmanager.yml

exec /bin/alertmanager "$@"
