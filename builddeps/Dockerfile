FROM scratch

ADD /rootfs.tar.gz /
COPY /nsproxy /nsproxy
COPY /remotemanager /remotemanager

CMD ["/nsproxy", "-chain"]
