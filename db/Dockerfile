FROM ghcr.io/amacneil/dbmate:2.37.0
COPY migrations /db/migrations
ENV DBMATE_MIGRATIONS_DIR=/db/migrations \
    DBMATE_NO_DUMP_SCHEMA=true
USER 10001
CMD ["--wait", "up"]
