#!/bin/sh

# the image ships a prebuilt /usr/bin/ghp-sync, so this only wires up cron and tails the log

#copy env
env >> /etc/profile

#setup cron
echo "$SYNC_CRON /app/scripts/run.sh 2>&1 | tee -a /var/log/ghp-sync.log" > /etc/cron.d/crontab
chmod 0644 /etc/cron.d/crontab
crontab /etc/cron.d/crontab
touch /var/log/cron.log

# start cron
/usr/sbin/crond -b
touch /var/log/ghp-sync.log
tail -f /var/log/ghp-sync.log
