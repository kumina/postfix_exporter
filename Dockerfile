FROM golang

COPY postfix_exporter /bin/postfix_exporter

RUN mkdir -p /var/spool/postfix/public/

RUN mkdir -p /var/log/

RUN touch /var/log/postfix_exporter_input.log

ENTRYPOINT /bin/postfix_exporter

EXPOSE 9154
