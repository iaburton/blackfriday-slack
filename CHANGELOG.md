# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2022-XX-XX
### Added
- Ability for `Renderer` to use other `io.Writers` more easily
- Error propegation from internal methods and rendering, usually those returned by `io.Writer`
- `Renderer` can be reused via `Reset`

### Changed
- The `Renderer` should no longer be made directly `r := &slackdown.Renderer{}`, instead it should be created 
using `NewRenderer`
- `blackfriday` updated to v2.1.0

### Fixed
- Moved package level state used by `Renderer` into `Renderer` itself, such that multiple can be used 
at the same time without issue.

## [0.1.0] - 2019-03-05
### Added
- go module support