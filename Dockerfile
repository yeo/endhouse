FROM debian

WORKDIR /endhouse

COPY --from=builder /app/endhouse /bin/

CMD ["/bin/endhouse"]
