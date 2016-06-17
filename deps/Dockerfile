FROM alpine

ADD rootfs.tar.gz /
COPY config.gcfg /
COPY nsproxy /
COPY run.sh /

CMD ["/run.sh"]
