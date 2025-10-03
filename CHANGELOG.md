# Changelog

All notable changes to this project will be documented in this file. See [commit-and-tag-version](https://github.com/absolute-version/commit-and-tag-version) for commit guidelines.

## [0.5.0](https://github.com/mikew/nvrh/compare/v0.4.0...v0.5.0) (2025-10-01)


### Features

* add Default config which applies to all servers ([#61](https://github.com/mikew/nvrh/issues/61)) ([ab63473](https://github.com/mikew/nvrh/commit/ab63473bf53ef35a2434307473d72d0a3218cf1b))
* replace remote directory's leading tilde with HOME variable ([#64](https://github.com/mikew/nvrh/issues/64)) ([e4c2710](https://github.com/mikew/nvrh/commit/e4c27109cf3cecfb623b2805d603766f7d7270ce))


### Bug Fixes

* Add pull_request trigger to ci ([ccfce59](https://github.com/mikew/nvrh/commit/ccfce596867747b056fe5bde9f3067cab96ecba9))

## [0.4.0](https://github.com/mikew/nvrh/compare/v0.3.0...v0.4.0) (2025-09-10)


### Features

* Config file support ([#58](https://github.com/mikew/nvrh/issues/58)) ([37e54ae](https://github.com/mikew/nvrh/commit/37e54ae41bc55dd995fa23ebce4e055796069587)), closes [#56](https://github.com/mikew/nvrh/issues/56)

## [0.3.0](https://github.com/mikew/nvrh/compare/v0.2.0...v0.3.0) (2025-08-20)


### Features

* Determine server info ([#52](https://github.com/mikew/nvrh/issues/52)) ([27c802b](https://github.com/mikew/nvrh/commit/27c802b1998c4cbca41abfc72a7c92517aa19506)), closes [#11](https://github.com/mikew/nvrh/issues/11)
* Windows terminal support ([#55](https://github.com/mikew/nvrh/issues/55)) ([f57e8a5](https://github.com/mikew/nvrh/commit/f57e8a5f5963760e9dcbdc07233ed554d5f5f6d0))


### Bug Fixes

* `client reconnect` command not getting server info ([#53](https://github.com/mikew/nvrh/issues/53)) ([2060a06](https://github.com/mikew/nvrh/commit/2060a06f6f36960ff5d260daa5fcdfe0c86adc20))
* `lua_files` -&gt; `bridge_files` due to shell scripts ([#48](https://github.com/mikew/nvrh/issues/48)) ([92e4e92](https://github.com/mikew/nvrh/commit/92e4e9249d80d95cc8023f151670075849b8945d))

## [0.2.0](https://github.com/mikew/nvrh/compare/v0.1.23...v0.2.0) (2025-08-15)


### Features

* Blur lines between "primary" and "secondary" ([#46](https://github.com/mikew/nvrh/issues/46)) ([ac6dc1c](https://github.com/mikew/nvrh/commit/ac6dc1c24719e648aabea221313b0f58995a1e55))
* Extract lua code ([#44](https://github.com/mikew/nvrh/issues/44)) ([3f7f3cc](https://github.com/mikew/nvrh/commit/3f7f3cc80ecee57cb0fdb200a671201ed027108e))
* No more primary/secondary ([#47](https://github.com/mikew/nvrh/issues/47)) ([737692f](https://github.com/mikew/nvrh/commit/737692f8b3b94a7476bb97ab8f9533af65d172ff))
* Switch to release-please ([#42](https://github.com/mikew/nvrh/issues/42)) ([1f86779](https://github.com/mikew/nvrh/commit/1f867797d184814175d06fd5561baab952379eff))
* Use `nvim_set_client_info` ([#45](https://github.com/mikew/nvrh/issues/45)) ([11431b3](https://github.com/mikew/nvrh/commit/11431b3c8da1cd63df58a97148d0f425d07733c6))

## 0.1.23 (2025-08-09)


### Features

* Better multiple sessions ([#41](https://github.com/mikew/nvrh/issues/41)) ([8f20913](https://github.com/mikew/nvrh/commit/8f20913c18d0d35542ab08aaa034ce9e961ca73f)), closes [#40](https://github.com/mikew/nvrh/issues/40)

## 0.1.22 (2025-08-07)


### Features

* Proper context / ctrl-c / timeout ([#39](https://github.com/mikew/nvrh/issues/39)) ([efcb7fd](https://github.com/mikew/nvrh/commit/efcb7fd19c866f1d099f711cb514d8c8124aded5)), closes [#38](https://github.com/mikew/nvrh/issues/38)

## 0.1.21 (2025-08-04)


### Features

* `--nvim-cmd`, `--enable-automap-ports`, and `--ssh-arg` ([#37](https://github.com/mikew/nvrh/issues/37)) ([ea961c3](https://github.com/mikew/nvrh/commit/ea961c34e128f6471a018aaca1f77a5740ebb022)), closes [#36](https://github.com/mikew/nvrh/issues/36)

## 0.1.20 (2025-08-01)


### Features

* Automatically map ports via scanning ([#31](https://github.com/mikew/nvrh/issues/31)) ([fe605f9](https://github.com/mikew/nvrh/commit/fe605f948978a67bae8b0bb78f6d2e1011f1e247)), closes [#10](https://github.com/mikew/nvrh/issues/10)

## 0.1.19 (2025-07-25)


### Features

* More env vars ([#32](https://github.com/mikew/nvrh/issues/32)) ([156d57e](https://github.com/mikew/nvrh/commit/156d57ed574194f46596c90c353bde775703b274))

## 0.1.18 (2025-07-25)


### Features

* Override vim.ui.open for http urls ([#35](https://github.com/mikew/nvrh/issues/35)) ([5649887](https://github.com/mikew/nvrh/commit/5649887a9711617ffbc1ca2584e2bbd70b803891)), closes [#34](https://github.com/mikew/nvrh/issues/34)

## 0.1.17 (2025-07-07)


### Features

* Move `cd`-ing to shell phase not vim phase ([#30](https://github.com/mikew/nvrh/issues/30)) ([0efff1b](https://github.com/mikew/nvrh/commit/0efff1ba260657c4f930a58f9802ddd376df2f2a))

## 0.1.16 (2025-07-07)


### Features

* Client reconnect ([#29](https://github.com/mikew/nvrh/issues/29)) ([9fc105d](https://github.com/mikew/nvrh/commit/9fc105d95308b8a69b8c96ec785c2a7401281207)), closes [#28](https://github.com/mikew/nvrh/issues/28)

## 0.1.15 (2024-11-17)


### Features

* Golang ssh ([#25](https://github.com/mikew/nvrh/issues/25)) ([228f5a0](https://github.com/mikew/nvrh/commit/228f5a0f839a842f515929250e9fe6f7f4309a05)), closes [#24](https://github.com/mikew/nvrh/issues/24)

## 0.1.14 (2024-10-21)


### Features

* Use loopback ([#26](https://github.com/mikew/nvrh/issues/26)) ([d5abb3a](https://github.com/mikew/nvrh/commit/d5abb3a3a205e3195a67e2edbcc222a593cc5466))

## 0.1.13 (2024-10-13)


### Bug Fixes

* Dont fork ssh ([#23](https://github.com/mikew/nvrh/issues/23)) ([ea4df7f](https://github.com/mikew/nvrh/commit/ea4df7f796a2e64913c2b88b08498a20daada23c))

## 0.1.12 (2024-10-12)


### Bug Fixes

* Same windows 255 fix for tunneling ports ([#22](https://github.com/mikew/nvrh/issues/22)) ([69c5739](https://github.com/mikew/nvrh/commit/69c57391293f79b76e26b08771b42c0c9c88b361))

## 0.1.11 (2024-10-12)


### Bug Fixes

* Incorrect socket path used browser script ([#21](https://github.com/mikew/nvrh/issues/21)) ([813d0d8](https://github.com/mikew/nvrh/commit/813d0d8c2027d9dee8a21be3d44a2113146b0235))

## 0.1.10 (2024-10-12)


### Features

* Cleanup processes ([#20](https://github.com/mikew/nvrh/issues/20)) ([944906f](https://github.com/mikew/nvrh/commit/944906f4ce91b6fb7806d72feecfad011e508d9b)), closes [#1](https://github.com/mikew/nvrh/issues/1)

## 0.1.9 (2024-10-11)


### Bug Fixes

* Strange 255 exit code on Windows ([#19](https://github.com/mikew/nvrh/issues/19)) ([438df45](https://github.com/mikew/nvrh/commit/438df4593cfe0097f36405e38bb77f090c51425b))

## 0.1.8 (2024-10-09)


### Features

* `--use-ports` option ([#18](https://github.com/mikew/nvrh/issues/18)) ([67431f7](https://github.com/mikew/nvrh/commit/67431f7014b0e131c7a8cabf84f21e06e46760e3)), closes [#17](https://github.com/mikew/nvrh/issues/17) [#2](https://github.com/mikew/nvrh/issues/2)

## 0.1.7 (2024-10-08)


### Bug Fixes

* `open-url` url guard and ssh `-t` flag ([#16](https://github.com/mikew/nvrh/issues/16)) ([b041de3](https://github.com/mikew/nvrh/commit/b041de32b589b12166c92f458373bc7b6eb447aa))

## 0.1.6 (2024-10-07)


### Features

* Add `--version` support ([#14](https://github.com/mikew/nvrh/issues/14)) ([5ca2a0a](https://github.com/mikew/nvrh/commit/5ca2a0a189123df443e8543b591770d7ca510b30))

## 0.1.5 (2024-10-07)

## 0.1.4 (2024-10-06)


### Bug Fixes

* Fix local editor flag ([#9](https://github.com/mikew/nvrh/issues/9)) ([b9e5cc3](https://github.com/mikew/nvrh/commit/b9e5cc3c1494b1bbebe45064b4b650125002ae8b))

## 0.1.3 (2024-10-06)

## 0.1.2 (2024-10-06)
