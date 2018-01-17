#!/usr/bin/env bash
ps cax | grep tpr-postman > /dev/null
if [ $? -eq 0 ]; then
  echo "Process is running."
else
  echo "Process is not running."
  /home/chris/tpr-gcloud/linux/tpr-postman distribute newsletter --email chris.witko@me.com
fi


