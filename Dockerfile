# syntax = basslang/frontend:dev

(use (*dir*/bass/bass-loop.bass))

(-> (from (linux/alpine)
      ($ cp (bass-loop:build *context*) /usr/local/bin/bass-loop))
    (with-entrypoint ["bass-loop"]))
