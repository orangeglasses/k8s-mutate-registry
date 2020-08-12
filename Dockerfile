FROM ubuntu

ENV PORT=8443

WORKDIR /home
COPY k8s-mutate-registry .

ENTRYPOINT /home/k8s-mutate-registry
#CMD /k8s-mutate-registry