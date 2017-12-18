FROM busybox:1.27-glibc

COPY postfix_exporter /bin/postfix_exporter

EXPOSE 9154
ENTRYPOINT [ "/bin/postfix_exporter" ]
