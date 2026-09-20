# check=skip=InvalidDefaultArgInFrom

# Go の版は workflow が mise.toml から読んで渡す。決め先を mise.toml だけに保つため
# ARG に既定値を置かず、それを咎める check を先頭で外している。
ARG GO_VERSION
FROM golang:${GO_VERSION} AS build
ARG VERSION
COPY . /src
# .git は同梱しないので、handshake が返す版は ldflags で埋める。
RUN CGO_ENABLED=0 go -C /src build -trimpath \
      -ldflags "-X github.com/boykush/adr/adi/cmd.version=${VERSION}" \
      -o /out/adi ./adi

# 配るのは決定そのものなので、決定も同梱する。adi はリクエストのたびにディスクから読む。
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/adi /usr/local/bin/adi
COPY decisions/ /adr/decisions/
# リポジトリと同じ配置にして --model を相対で渡す。get_decision が返す path が、
# このリポジトリを clone したクライアントから見たパスと揃う。
WORKDIR /adr
EXPOSE 8080
# 呼び出し側が変えるのは listen アドレスだけなので、それを唯一の引数にする。
# distroless には shell が無く、環境変数から展開できない。
ENTRYPOINT ["/usr/local/bin/adi", "mcp", "--model", "decisions", "--http"]
CMD ["0.0.0.0:8080"]
