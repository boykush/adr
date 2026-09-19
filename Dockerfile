# check=skip=InvalidDefaultArgInFrom

# fork はリリースを切っていないので、mise.toml の ref で clone してビルドする。
# Go と fork の版は workflow が mise.toml から読んで渡す。決め先を mise.toml だけに
# 保つため ARG に既定値を置かず、それを咎める check を先頭で外している。
ARG GO_VERSION
FROM golang:${GO_VERSION} AS build
ARG ADG_REF
RUN git clone --quiet https://github.com/boykush/ad-guidance-tool.git /src \
 && git -C /src checkout --quiet "${ADG_REF}" \
 && CGO_ENABLED=0 go -C /src build -trimpath -o /out/adg ./adg

# 配るのは決定そのものなので、model も同梱する。adg はリクエストのたびにディスクから読む。
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/adg /usr/local/bin/adg
COPY decisions/ /adr/decisions/
# リポジトリと同じ配置にして --model を相対で渡す。get_adr が返すパスが、
# このリポジトリを clone したクライアントから見たパス（decisions/…）と揃う。
WORKDIR /adr
EXPOSE 8080
# 呼び出し側が変えるのは listen アドレスだけなので、それを唯一の引数にする。
# distroless には shell が無く、環境変数から展開できない。
ENTRYPOINT ["/usr/local/bin/adg", "mcp", "run", "--model", "decisions", "--http"]
CMD ["0.0.0.0:8080"]
