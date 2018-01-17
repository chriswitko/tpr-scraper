#!/usr/bin/env bash
ps cax | grep tpr-journalist > /dev/null
if [ $? -eq 0 ]; then
  echo "Process is running."
else
  echo "Process is not running."
  /home/chris/tpr-gcloud/linux/tpr-journalist collect --all --log --save
fi
