# Changelog

## [1.7.9](https://github.com/ship-digital/pull-watch/compare/v1.7.8...v1.7.9) (2025-04-11)


### Bug Fixes

* **publish-npm:** add RELEASE_TAG support for version determination ([#50](https://github.com/ship-digital/pull-watch/issues/50)) ([b103d2f](https://github.com/ship-digital/pull-watch/commit/b103d2fc6fddea68372cfb2636865d186a7a913a))

## [1.7.8](https://github.com/ship-digital/pull-watch/compare/v1.7.7...v1.7.8) (2025-04-11)


### Bug Fixes

* simplify release command in workflow ([#48](https://github.com/ship-digital/pull-watch/issues/48)) ([10163af](https://github.com/ship-digital/pull-watch/commit/10163af95d94899e4e574084d062967f6fd1b65c))

## [1.7.7](https://github.com/ship-digital/pull-watch/compare/v1.7.6...v1.7.7) (2025-04-11)


### Bug Fixes

* specify shell for tag determination step in release workflow ([#46](https://github.com/ship-digital/pull-watch/issues/46)) ([0ea9f49](https://github.com/ship-digital/pull-watch/commit/0ea9f4982c8c0f13037df9f5a08f725538839095))

## [1.7.6](https://github.com/ship-digital/pull-watch/compare/v1.7.5...v1.7.6) (2025-04-11)


### Bug Fixes

* pin GoReleaser version in workflow ([#44](https://github.com/ship-digital/pull-watch/issues/44)) ([4586237](https://github.com/ship-digital/pull-watch/commit/45862370b83f62522810e31cf261e494e5be3b06))

## [1.7.5](https://github.com/ship-digital/pull-watch/compare/v1.7.4...v1.7.5) (2025-04-11)


### Bug Fixes

* remove goreleaser workflow file ([#42](https://github.com/ship-digital/pull-watch/issues/42)) ([71d77f3](https://github.com/ship-digital/pull-watch/commit/71d77f300035f688371ac9026c40f0462cbce3c4))

## [1.7.4](https://github.com/ship-digital/pull-watch/compare/v1.7.3...v1.7.4) (2025-04-11)


### Bug Fixes

* update GoReleaser workflow to include NPM_TOKEN in environment variables ([#40](https://github.com/ship-digital/pull-watch/issues/40)) ([35d77c9](https://github.com/ship-digital/pull-watch/commit/35d77c96a1f78f73e0255afa98c67932fad5326b))

## [1.7.1](https://github.com/ship-digital/pull-watch/compare/v1.7.0...v1.7.1) (2025-04-11)


### Bug Fixes

* trigger patch release ([cc5632f](https://github.com/ship-digital/pull-watch/commit/cc5632f13a4e59bc97d9e948fa92752844371e46))

## [1.7.0](https://github.com/ship-digital/pull-watch/compare/v1.6.0...v1.7.0) (2025-04-08)


### Features

* npm package ([#35](https://github.com/ship-digital/pull-watch/issues/35)) ([c5db6c0](https://github.com/ship-digital/pull-watch/commit/c5db6c0fcb72699c9c6a7a21ddd9e596152973b4))

## [1.6.0](https://github.com/ship-digital/pull-watch/compare/v1.5.4...v1.6.0) (2025-04-06)


### Features

* no-restart ([#33](https://github.com/ship-digital/pull-watch/issues/33)) ([ffa51fd](https://github.com/ship-digital/pull-watch/commit/ffa51fdb6eea9ff36d145b9b8e5291754a06f752))

## [1.5.4](https://github.com/ship-digital/pull-watch/compare/v1.5.3...v1.5.4) (2024-12-22)


### Bug Fixes

* Update winget package name in .goreleaser.yaml ([f59b5eb](https://github.com/ship-digital/pull-watch/commit/f59b5eb835c7c4da34984c65c133550fa000e68e))

## [1.5.3](https://github.com/ship-digital/pull-watch/compare/v1.5.2...v1.5.3) (2024-12-22)


### Bug Fixes

* Correct publisher name in winget configuration ([eea67b4](https://github.com/ship-digital/pull-watch/commit/eea67b486a38069e98f147a38c00b66019c64950))

## [1.5.2](https://github.com/ship-digital/pull-watch/compare/v1.5.1...v1.5.2) (2024-12-22)


### Bug Fixes

* Update winget configuration in goreleaser ([bd019d1](https://github.com/ship-digital/pull-watch/commit/bd019d1078789f7ae35a78277e254fca58465f4d))

## [1.5.1](https://github.com/ship-digital/pull-watch/compare/v1.5.0...v1.5.1) (2024-12-22)


### Bug Fixes

* winget automated pull-request ([ab2a4e2](https://github.com/ship-digital/pull-watch/commit/ab2a4e20cbba87786093de1f9f4a3d263dfe0155))

## [1.5.0](https://github.com/ship-digital/pull-watch/compare/v1.4.0...v1.5.0) (2024-12-22)


### Features

* Add version flag to MainCommand and update README ([0f8eae0](https://github.com/ship-digital/pull-watch/commit/0f8eae065668afcba3b9df396e4194e6ce59d468))

## [1.4.0](https://github.com/ship-digital/pull-watch/compare/v1.3.3...v1.4.0) (2024-12-11)


### Features

* Enhance process management with PID tracking and error handling ([#25](https://github.com/ship-digital/pull-watch/issues/25)) ([49fa8a4](https://github.com/ship-digital/pull-watch/commit/49fa8a49846b27b5c6ea845f9d8ac8f5de42db4d))

## [1.3.3](https://github.com/ship-digital/pull-watch/compare/v1.3.2...v1.3.3) (2024-12-11)


### Bug Fixes

* Enhance commit existence checks in GitRepository ([#23](https://github.com/ship-digital/pull-watch/issues/23)) ([4aea4b3](https://github.com/ship-digital/pull-watch/commit/4aea4b3044dd4f5c1b8b3d3002616f9e2a1f27ef))

## [1.3.2](https://github.com/ship-digital/pull-watch/compare/v1.3.1...v1.3.2) (2024-12-11)


### Bug Fixes

* 🐛 Update version output in VersionCommand to include build information ([b8eec3c](https://github.com/ship-digital/pull-watch/commit/b8eec3c896997059bf3c03d1b016e4d30b69cdc5))

## [1.3.1](https://github.com/ship-digital/pull-watch/compare/v1.3.0...v1.3.1) (2024-12-11)


### Bug Fixes

* 🐛 Add IsRunning method to ProcessManager for improved process state management ([f2d2c95](https://github.com/ship-digital/pull-watch/commit/f2d2c95574b58c67df97b0321e5714a23bbaad9a))

## [1.3.0](https://github.com/ship-digital/pull-watch/compare/v1.2.0...v1.3.0) (2024-12-11)


### Features

* ✨ Enhance logging and command handling in pull-watch ([4ec0ac0](https://github.com/ship-digital/pull-watch/commit/4ec0ac0631a730816813f0f40d5c829680a6dbe7))

## [1.2.0](https://github.com/ship-digital/pull-watch/compare/v1.1.0...v1.2.0) (2024-12-11)


### Features

* 🎉 Introduce main application logic and update build configuration ([05835f8](https://github.com/ship-digital/pull-watch/commit/05835f8524a4aa20037e15da766ef0a3a36953d1))

## [1.1.0](https://github.com/ship-digital/pull-watch/compare/v1.0.2...v1.1.0) (2024-12-09)


### Features

* 🚀Enhance pull-watch functionality and documentation ([#8](https://github.com/ship-digital/pull-watch/issues/8)) ([372feb6](https://github.com/ship-digital/pull-watch/commit/372feb6470991e3792ce1bba10b7eb9b4df6a339))

## [1.0.2](https://github.com/ship-digital/pull-watch/compare/v1.0.1...v1.0.2) (2024-12-09)


### Bug Fixes

* get latest remote changes before running the command for the first time ([#7](https://github.com/ship-digital/pull-watch/issues/7)) ([9de0ee4](https://github.com/ship-digital/pull-watch/commit/9de0ee4cde0c66fd57e9835e899bb98176fdd49f))
