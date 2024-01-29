FROM scratch
EXPOSE 9154
COPY postfix_exporter /
COPY LICENSE /
ENTRYPOINT ["/postfix_exporter"]
