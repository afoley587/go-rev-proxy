FROM nginxdemos/hello

COPY override-entrypoint.sh /override-entrypoint.sh

RUN chmod +x /override-entrypoint.sh

ENTRYPOINT [ "/override-entrypoint.sh" ]