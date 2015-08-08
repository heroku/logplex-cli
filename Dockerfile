FROM scratch

# copy all static binaries
COPY dist/logplex-cli /logplex-cli

CMD ["/logplex-cli"]
