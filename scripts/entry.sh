#!/bin/sh

# the image ships a prebuilt /usr/bin/ghp-sync, so this only wires up cron and tails the log

#copy env
env >> /etc/profile

#setup cron: the file stays out of /etc/cron.d, which dcron reads on its own - a crontab written
# there and then installed with crontab is registered twice, and every sync runs twice at once
echo "$SYNC_CRON /app/scripts/run.sh 2>&1 | tee -a /var/log/ghp-sync.log" > /app/crontab
crontab /app/crontab
touch /var/log/cron.log

# start cron
/usr/sbin/crond -b
touch /var/log/ghp-sync.log
tail -f /var/log/ghp-sync.log
