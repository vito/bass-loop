# syntax = basslang/frontend:0.12.0
(use (.git (linux/alpine/git))
     (git:github/vito/tabs/ref/main/nix.bass)
     (*dir*/bass/bass-loop.bass))

(-> (from (nix:linux/busybox :flake *dir*)
      ($ cp (bass-loop:build *dir*) /usr/bin/bass-loop))
    (with-port :http 3000)
    (with-port :ssh 6455)
    (with-entrypoint ["bass-loop" "--listen" "0.0.0.0:3000"]))
