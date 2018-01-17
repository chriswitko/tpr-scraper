FROM almir/webhook
LABEL author="Chris Witko <chris.witko@gmail.com>"
RUN apk add --no-cache bash gawk sed grep bc coreutils
# VOLUME [ ".:/etc/webhook" ]
# WORKDIR /etc/webhook

COPY build/linux/journalist /var/scripts/journalist
COPY build/linux/.env /var/scripts/.env
COPY ./run-journalist.sh /var/scripts/run-journalist.sh

RUN mkdir /var/scripts/tmp
RUN chmod +x /var/scripts/run-journalist.sh

COPY hooks.json /etc/webhook/hooks.json
CMD ["-verbose", "-hooks=/etc/webhook/hooks.json", "-hotreload"]